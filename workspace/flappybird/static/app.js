/**
 * Harness Cyber-Flap — Frontend Game Engine & View Controller
 * Implements FSM state transitions, Canvas rendering, procedural Web Audio SFX,
 * AABB collision detection, and delta-time normalized physics.
 */

// Game Configuration matching Go engine constants
const ScreenWidth = 480.0;
const ScreenHeight = 640.0;
const GroundY = 560.0;
const CeilingY = 0.0;
const BirdWidth = 34.0;
const BirdHeight = 24.0;
const BirdStartX = 100.0;
const BirdStartY = 240.0;
const PipeWidth = 52.0;
const PipeGap = 120.0;
const MinPipeHeight = 50.0;
const MaxPipeHeight = 350.0;
const Gravity = 800.0;
const FlapImpulse = -250.0;
const HorizontalSpeed = 120.0;
const PipeSpawnInterval = 1.5;
const FloatingAmplitude = 8.0;
const FloatingSpeed = 4.0;

// Game States
const StateStartScreen = 0;
const StatePlaying = 1;
const StateGameOver = 2;

class AudioSynthEngine {
    constructor() {
        this.ctx = null;
        this.enabled = true;
    }

    init() {
        if (!this.ctx) {
            this.ctx = new (window.AudioContext || window.webkitAudioContext)();
        }
    }

    playFlap() {
        if (!this.enabled) return;
        this.init();
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();
        
        osc.type = 'triangle';
        osc.frequency.setValueAtTime(150, this.ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(450, this.ctx.currentTime + 0.12);
        
        gain.gain.setValueAtTime(0.2, this.ctx.currentTime);
        gain.gain.linearRampToValueAtTime(0.01, this.ctx.currentTime + 0.12);
        
        osc.connect(gain);
        gain.connect(this.ctx.destination);
        osc.start();
        osc.stop(this.ctx.currentTime + 0.12);
    }

    playScore() {
        if (!this.enabled) return;
        this.init();
        const osc1 = this.ctx.createOscillator();
        const osc2 = this.ctx.createOscillator();
        const gain = this.ctx.createGain();
        
        osc1.type = 'sine';
        osc2.type = 'sine';
        
        // Classic retro coin chime: two rapid tones
        osc1.frequency.setValueAtTime(523.25, this.ctx.currentTime); // C5
        osc1.frequency.setValueAtTime(659.25, this.ctx.currentTime + 0.08); // E5
        
        gain.gain.setValueAtTime(0.15, this.ctx.currentTime);
        gain.gain.linearRampToValueAtTime(0.01, this.ctx.currentTime + 0.25);
        
        osc1.connect(gain);
        gain.connect(this.ctx.destination);
        osc1.start();
        osc1.stop(this.ctx.currentTime + 0.25);
    }

    playHit() {
        if (!this.enabled) return;
        this.init();
        
        // Noise buffer for impact sound
        const bufferSize = this.ctx.sampleRate * 0.15;
        const buffer = this.ctx.createBuffer(1, bufferSize, this.ctx.sampleRate);
        const data = buffer.getChannelData(0);
        for (let i = 0; i < bufferSize; i++) {
            data[i] = Math.random() * 2 - 1;
        }
        
        const noiseNode = this.ctx.createBufferSource();
        noiseNode.buffer = buffer;
        
        const filter = this.ctx.createBiquadFilter();
        filter.type = 'lowpass';
        filter.frequency.setValueAtTime(400, this.ctx.currentTime);
        filter.frequency.exponentialRampToValueAtTime(10, this.ctx.currentTime + 0.15);
        
        const gain = this.ctx.createGain();
        gain.gain.setValueAtTime(0.3, this.ctx.currentTime);
        gain.gain.linearRampToValueAtTime(0.01, this.ctx.currentTime + 0.15);
        
        noiseNode.connect(filter);
        filter.connect(gain);
        gain.connect(this.ctx.destination);
        
        noiseNode.start();
        noiseNode.stop(this.ctx.currentTime + 0.15);
    }

    playGameOver() {
        if (!this.enabled) return;
        this.init();
        
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();
        
        osc.type = 'sawtooth';
        osc.frequency.setValueAtTime(300, this.ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(60, this.ctx.currentTime + 0.6);
        
        gain.gain.setValueAtTime(0.2, this.ctx.currentTime);
        gain.gain.linearRampToValueAtTime(0.01, this.ctx.currentTime + 0.6);
        
        osc.connect(gain);
        gain.connect(this.ctx.destination);
        
        osc.start();
        osc.stop(this.ctx.currentTime + 0.6);
    }
}

class GameEngine {
    constructor(canvas, startOverlay, gameOverOverlay) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.startOverlay = startOverlay;
        this.gameOverOverlay = gameOverOverlay;
        this.audio = new AudioSynthEngine();
        
        // Game variables
        this.state = StateStartScreen;
        this.birdY = BirdStartY;
        this.birdVelocityY = 0.0;
        this.pipes = [];
        this.score = 0;
        this.highScore = parseInt(localStorage.getItem('harness_high_score') || '0', 10);
        
        this.spawnTimer = 0.0;
        this.runningTime = 0.0;
        
        // Loop telemetry
        this.lastTime = 0;
        this.fpsCount = 0;
        this.fpsTimer = 0;
        this.fps = 60;
        
        // Graphic elements
        this.stars = [];
        this.particles = [];
        this.generateStars();
        
        // Event Listeners setup
        this.bindEvents();
    }
    
    generateStars() {
        this.stars = [];
        for (let i = 0; i < 40; i++) {
            this.stars.push({
                x: Math.random() * ScreenWidth,
                y: Math.random() * GroundY,
                size: Math.random() * 2 + 0.5,
                speed: (Math.random() * 0.4 + 0.1) * 30 // relative to HorizontalSpeed
            });
        }
    }
    
    spawnParticle(x, y, color) {
        for (let i = 0; i < 8; i++) {
            this.particles.push({
                x: x,
                y: y,
                vx: (Math.random() - 0.5) * 80 - 40,
                vy: (Math.random() - 0.5) * 80 - 20,
                size: Math.random() * 3 + 1,
                alpha: 1.0,
                color: color
            });
        }
    }
    
    bindEvents() {
        this.flapHandler = (e) => {
            if (e.code === 'Space') {
                e.preventDefault();
                this.flap();
            }
        };
        
        this.canvasHandler = (e) => {
            e.preventDefault();
            this.flap();
        };
        
        window.addEventListener('keydown', this.flapHandler);
        this.canvas.addEventListener('mousedown', this.canvasHandler);
        this.canvas.addEventListener('touchstart', this.canvasHandler, { passive: false });
        
        // Audio toggle hook
        const audioToggle = document.getElementById('audioToggle');
        if (audioToggle) {
            audioToggle.addEventListener('change', (e) => {
                this.audio.enabled = e.target.checked;
            });
        }
    }
    
    cleanupEvents() {
        window.removeEventListener('keydown', this.flapHandler);
        this.canvas.removeEventListener('mousedown', this.canvasHandler);
        this.canvas.removeEventListener('touchstart', this.canvasHandler);
    }
    
    reset() {
        this.birdY = BirdStartY;
        this.birdVelocityY = 0.0;
        this.pipes = [];
        this.score = 0;
        this.spawnTimer = 0.0;
        this.runningTime = 0.0;
        this.particles = [];
        this.state = StateStartScreen;
        
        this.startOverlay.classList.add('active');
        this.gameOverOverlay.classList.remove('active');
    }
    
    flap() {
        if (this.state === StateStartScreen) {
            this.startOverlay.classList.remove('active');
            this.state = StatePlaying;
            this.birdVelocityY = FlapImpulse;
            this.audio.playFlap();
            this.spawnParticle(BirdStartX, this.birdY, '#00f2fe');
        } else if (this.state === StatePlaying) {
            this.birdVelocityY = FlapImpulse;
            this.audio.playFlap();
            this.spawnParticle(BirdStartX, this.birdY, '#00f2fe');
        }
    }
    
    startLoop() {
        this.lastTime = performance.now();
        const loop = (timestamp) => {
            let dt = (timestamp - this.lastTime) / 1000.0;
            this.lastTime = timestamp;
            
            // Limit frame step during background tabs
            if (dt > 0.1) dt = 0.1;
            
            this.update(dt);
            this.draw();
            
            // Measure performance FPS
            this.fpsCount++;
            this.fpsTimer += dt;
            if (this.fpsTimer >= 1.0) {
                this.fps = this.fpsCount;
                document.getElementById('fps-val').innerText = this.fps;
                this.fpsCount = 0;
                this.fpsTimer = 0;
            }
            
            requestAnimationFrame(loop);
        };
        requestAnimationFrame(loop);
    }
    
    update(dt) {
        if (dt <= 0.0) return;
        
        // Scroll stars in background parallax
        this.stars.forEach(star => {
            star.x -= star.speed * dt;
            if (star.x < 0) {
                star.x = ScreenWidth;
            }
        });
        
        // Update particles
        this.particles.forEach(p => {
            p.x += p.vx * dt;
            p.y += p.vy * dt;
            p.alpha -= 1.5 * dt;
        });
        this.particles = this.particles.filter(p => p.alpha > 0);
        
        if (this.state === StateStartScreen) {
            this.runningTime += dt;
            // Floating animation matching Go engine
            this.birdY = BirdStartY + FloatingAmplitude * Math.sin(FloatingSpeed * this.runningTime);
            this.birdVelocityY = 0.0;
        } else if (this.state === StatePlaying) {
            this.runningTime += dt;
            
            // Apply physics gravity
            this.birdVelocityY += Gravity * dt;
            this.birdY += this.birdVelocityY * dt;
            
            // Ceiling and Ground Collisions
            if (this.birdY - BirdHeight / 2 <= CeilingY) {
                this.birdY = CeilingY + BirdHeight / 2;
                this.handleCollision();
                return;
            }
            if (this.birdY + BirdHeight / 2 >= GroundY) {
                this.birdY = GroundY - BirdHeight / 2;
                this.handleCollision();
                return;
            }
            
            // Pipe generation
            this.spawnTimer += dt;
            if (this.spawnTimer >= PipeSpawnInterval) {
                this.spawnTimer -= PipeSpawnInterval;
                this.spawnPipePair();
            }
            
            // Update pipes and check collisions
            this.pipes.forEach(pipe => {
                pipe.x -= HorizontalSpeed * dt;
                
                // Check collision
                if (this.checkCollision(pipe)) {
                    this.handleCollision();
                    return;
                }
                
                // Check score
                if (!pipe.scored && (BirdStartX - BirdWidth / 2) > (pipe.x + PipeWidth)) {
                    pipe.scored = true;
                    this.score++;
                    this.audio.playScore();
                    this.spawnParticle(BirdStartX, this.birdY, '#ff007f');
                    
                    if (this.score > this.highScore) {
                        this.highScore = this.score;
                        localStorage.setItem('harness_high_score', this.highScore);
                    }
                }
            });
            
            // Clean up off-screen pipes
            this.pipes = this.pipes.filter(pipe => (pipe.x + PipeWidth) >= 0);
        }
    }
    
    spawnPipePair() {
        let maxTopHeight = GroundY - PipeGap - MinPipeHeight;
        if (maxTopHeight > MaxPipeHeight) {
            maxTopHeight = MaxPipeHeight;
        }
        const heightRange = maxTopHeight - MinPipeHeight;
        let topHeight = MinPipeHeight;
        if (heightRange > 0) {
            topHeight += Math.random() * heightRange;
        }
        let bottomHeight = GroundY - topHeight - PipeGap;
        
        this.pipes.push({
            x: ScreenWidth,
            topHeight: topHeight,
            bottomHeight: bottomHeight,
            scored: false
        });
    }
    
    checkCollision(pipe) {
        const birdLeft = BirdStartX - BirdWidth / 2;
        const birdRight = BirdStartX + BirdWidth / 2;
        const birdTop = this.birdY - BirdHeight / 2;
        const birdBottom = this.birdY + BirdHeight / 2;
        
        const pipeLeft = pipe.x;
        const pipeRight = pipe.x + PipeWidth;
        
        // AABB Horizontal checks
        if (birdRight < pipeLeft || birdLeft > pipeRight) {
            return false;
        }
        
        // Top pipe overlap
        if (birdTop < pipe.topHeight) {
            return true;
        }
        
        // Bottom pipe overlap
        const bottomPipeTopY = GroundY - pipe.bottomHeight;
        if (birdBottom > bottomPipeTopY) {
            return true;
        }
        
        return false;
    }
    
    handleCollision() {
        this.state = StateGameOver;
        this.audio.playHit();
        this.audio.playGameOver();
        this.spawnParticle(BirdStartX, this.birdY, '#ff3838');
        
        document.getElementById('finalScore').innerText = this.score;
        document.getElementById('highScore').innerText = this.highScore;
        this.gameOverOverlay.classList.add('active');
    }
    
    draw() {
        this.ctx.clearRect(0, 0, ScreenWidth, ScreenHeight);
        
        // 1. Draw Starry Night Background
        this.ctx.fillStyle = '#05020c';
        this.ctx.fillRect(0, 0, ScreenWidth, ScreenHeight);
        
        this.ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
        this.stars.forEach(star => {
            this.ctx.beginPath();
            this.ctx.arc(star.x, star.y, star.size, 0, Math.PI * 2);
            this.ctx.fill();
        });
        
        // 2. Draw Distant Synthwave Grid Horizon
        this.ctx.strokeStyle = 'rgba(255, 0, 127, 0.1)';
        this.ctx.lineWidth = 1;
        for (let i = 0; i < ScreenWidth; i += 40) {
            this.ctx.beginPath();
            this.ctx.moveTo(i, 400);
            this.ctx.lineTo(i + (i - ScreenWidth/2)*1.5, GroundY);
            this.ctx.stroke();
        }
        for (let i = 400; i < GroundY; i += 30) {
            this.ctx.beginPath();
            this.ctx.moveTo(0, i);
            this.ctx.lineTo(ScreenWidth, i);
            this.ctx.stroke();
        }
        
        // 3. Draw Pipes (Cyber/Neon styling)
        this.pipes.forEach(pipe => {
            // Draw top pipe
            const topGrad = this.ctx.createLinearGradient(pipe.x, 0, pipe.x + PipeWidth, 0);
            topGrad.addColorStop(0, '#00f2fe');
            topGrad.addColorStop(0.5, '#0072ff');
            topGrad.addColorStop(1, '#00f2fe');
            
            this.ctx.fillStyle = topGrad;
            this.ctx.fillRect(pipe.x, 0, PipeWidth, pipe.topHeight);
            
            // Glow effect for pipe borders
            this.ctx.shadowBlur = 8;
            this.ctx.shadowColor = '#00f2fe';
            this.ctx.strokeStyle = '#00f2fe';
            this.ctx.lineWidth = 2;
            this.ctx.strokeRect(pipe.x, 0, PipeWidth, pipe.topHeight);
            
            // Pipe cap top
            this.ctx.fillStyle = '#030107';
            this.ctx.fillRect(pipe.x - 2, pipe.topHeight - 20, PipeWidth + 4, 20);
            this.ctx.strokeRect(pipe.x - 2, pipe.topHeight - 20, PipeWidth + 4, 20);
            
            // Draw bottom pipe
            const bottomPipeY = GroundY - pipe.bottomHeight;
            const bottomGrad = this.ctx.createLinearGradient(pipe.x, bottomPipeY, pipe.x + PipeWidth, bottomPipeY);
            bottomGrad.addColorStop(0, '#00f2fe');
            bottomGrad.addColorStop(0.5, '#0072ff');
            bottomGrad.addColorStop(1, '#00f2fe');
            
            this.ctx.fillStyle = bottomGrad;
            this.ctx.fillRect(pipe.x, bottomPipeY, PipeWidth, pipe.bottomHeight);
            this.ctx.strokeRect(pipe.x, bottomPipeY, PipeWidth, pipe.bottomHeight);
            
            // Pipe cap bottom
            this.ctx.fillStyle = '#030107';
            this.ctx.fillRect(pipe.x - 2, bottomPipeY, PipeWidth + 4, 20);
            this.ctx.strokeRect(pipe.x - 2, bottomPipeY, PipeWidth + 4, 20);
            
            // Reset shadows
            this.ctx.shadowBlur = 0;
        });
        
        // 4. Draw Particles
        this.particles.forEach(p => {
            this.ctx.save();
            this.ctx.globalAlpha = p.alpha;
            this.ctx.fillStyle = p.color;
            this.ctx.shadowBlur = 8;
            this.ctx.shadowColor = p.color;
            this.ctx.fillRect(p.x, p.y, p.size, p.size);
            this.ctx.restore();
        });
        
        // 5. Draw Cyber-Bird (Cyberpunk Spaceship design)
        this.ctx.save();
        this.ctx.translate(BirdStartX, this.birdY);
        
        // Rotation based on vertical velocity
        let angle = Math.atan2(this.birdVelocityY, HorizontalSpeed * 1.5);
        this.ctx.rotate(angle);
        
        // Shadow glow
        this.ctx.shadowBlur = 12;
        this.ctx.shadowColor = '#00f2fe';
        
        // Engine fire flame if active or flapping
        if (this.state === StatePlaying && this.birdVelocityY < 0) {
            const fireGrad = this.ctx.createLinearGradient(-30, 0, -17, 0);
            fireGrad.addColorStop(0, 'rgba(255, 0, 127, 0)');
            fireGrad.addColorStop(0.5, '#ff007f');
            fireGrad.addColorStop(1, '#ffeb3b');
            this.ctx.fillStyle = fireGrad;
            this.ctx.beginPath();
            this.ctx.moveTo(-17, -6);
            this.ctx.lineTo(-30 - Math.random() * 10, 0);
            this.ctx.lineTo(-17, 6);
            this.ctx.closePath();
            this.ctx.fill();
        }
        
        // Ship Hull
        this.ctx.fillStyle = '#120a24';
        this.ctx.strokeStyle = '#00f2fe';
        this.ctx.lineWidth = 2;
        
        this.ctx.beginPath();
        this.ctx.moveTo(17, 0);       // Nose cone
        this.ctx.lineTo(-10, -12);    // Top wing edge
        this.ctx.lineTo(-17, -10);    // Top rear
        this.ctx.lineTo(-12, 0);      // Middle rear indent
        this.ctx.lineTo(-17, 10);     // Bottom rear
        this.ctx.lineTo(-10, 12);     // Bottom wing edge
        this.ctx.closePath();
        this.ctx.fill();
        this.ctx.stroke();
        
        // Glowing cockpit dome
        this.ctx.fillStyle = '#ff007f';
        this.ctx.shadowColor = '#ff007f';
        this.ctx.beginPath();
        this.ctx.arc(3, -2, 4, 0, Math.PI * 2);
        this.ctx.fill();
        
        this.ctx.restore();
        
        // 6. Draw Ground Layer
        const groundGrad = this.ctx.createLinearGradient(0, GroundY, 0, ScreenHeight);
        groundGrad.addColorStop(0, '#120a24');
        groundGrad.addColorStop(1, '#05020c');
        this.ctx.fillStyle = groundGrad;
        this.ctx.fillRect(0, GroundY, ScreenWidth, ScreenHeight - GroundY);
        
        this.ctx.strokeStyle = '#ff007f';
        this.ctx.lineWidth = 3;
        this.ctx.shadowBlur = 10;
        this.ctx.shadowColor = '#ff007f';
        this.ctx.beginPath();
        this.ctx.moveTo(0, GroundY);
        this.ctx.lineTo(ScreenWidth, GroundY);
        this.ctx.stroke();
        this.ctx.shadowBlur = 0; // Reset
        
        // 7. Score HUD (Only if playing)
        if (this.state === StatePlaying) {
            this.ctx.font = '900 2.2rem "Orbitron", sans-serif';
            this.ctx.fillStyle = '#fff';
            this.ctx.textAlign = 'center';
            
            this.ctx.shadowBlur = 8;
            this.ctx.shadowColor = '#00f2fe';
            this.ctx.fillText(this.score, ScreenWidth / 2, 80);
            this.ctx.shadowBlur = 0;
        }
    }
}

// Instantiate and initiate execution flow on document load
document.addEventListener('DOMContentLoaded', () => {
    const canvas = document.getElementById('gameCanvas');
    const startOverlay = document.getElementById('startOverlay');
    const gameOverOverlay = document.getElementById('gameOverOverlay');
    const startButton = document.getElementById('startButton');
    const restartButton = document.getElementById('restartButton');
    
    const game = new GameEngine(canvas, startOverlay, gameOverOverlay);
    
    // Performance latency simulation tracker
    let lastStamp = performance.now();
    setInterval(() => {
        const currentStamp = performance.now();
        const latency = currentStamp - lastStamp - 100.0;
        lastStamp = currentStamp;
        document.getElementById('latency-val').innerText = `${Math.max(0.1, latency).toFixed(1)}ms`;
    }, 100);
    
    startButton.addEventListener('click', () => {
        game.flap();
    });
    
    restartButton.addEventListener('click', () => {
        game.reset();
    });
    
    game.startLoop();
});
