/**
 * Harness Engineering Landing Page Interactive Logic
 * Exposes fluid micro-interactions, responsive form verification,
 * scroll animation reveals, and a live pipeline orchestration simulator.
 */

document.addEventListener('DOMContentLoaded', () => {
  // ─────────────────────────────────────────────
  // 1. STICKY HEADER NAVIGATION
  // ─────────────────────────────────────────────
  const header = document.querySelector('header');
  const scrollThreshold = 50;

  window.addEventListener('scroll', () => {
    if (window.scrollY > scrollThreshold) {
      header.classList.add('scrolled');
    } else {
      header.classList.remove('scrolled');
    }
    updateActiveNav();
  });

  // ─────────────────────────────────────────────
  // 2. ACTIVE NAVIGATION STATE TRACKER
  // ─────────────────────────────────────────────
  const sections = document.querySelectorAll('section');
  const navLinks = document.querySelectorAll('nav a');

  function updateActiveNav() {
    let currentId = '';
    const scrollPos = window.scrollY + 100; // Account for header offset

    sections.forEach(section => {
      const top = section.offsetTop;
      const height = section.offsetHeight;
      if (scrollPos >= top && scrollPos < top + height) {
        currentId = section.getAttribute('id');
      }
    });

    navLinks.forEach(link => {
      link.classList.remove('active');
      if (link.getAttribute('href') === `#${currentId}`) {
        link.classList.add('active');
      }
    });
  }

  // ─────────────────────────────────────────────
  // 3. SCROLL REVEAL (INTERSECTION OBSERVER)
  // ─────────────────────────────────────────────
  const revealElements = document.querySelectorAll('.reveal');
  
  if ('IntersectionObserver' in window) {
    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          entry.target.classList.add('active');
          observer.unobserve(entry.target); // Reveal once
        }
      });
    }, {
      threshold: 0.15
    });

    revealElements.forEach(el => observer.observe(el));
  } else {
    // Fallback if IntersectionObserver is unsupported
    revealElements.forEach(el => el.classList.add('active'));
  }

  // ─────────────────────────────────────────────
  // 4. LIVE PIPELINE ORCHESTRATION SIMULATOR
  // ─────────────────────────────────────────────
  const pipelineNodes = document.querySelectorAll('.pipeline-node');
  const pipelineStateText = document.querySelector('.hero-badge span');
  const pipelineStates = [
    { name: 'DEV_CODING', text: 'Pipeline Status: Synthesizing validation modules...', status: 'Synthesizing password hashing code...' },
    { name: 'QA_TESTING', text: 'Pipeline Status: Running Go verification suite...', status: 'Checking bcrypt limits (72B, cost: 10)...' },
    { name: 'DEVOPS_DELIVER', text: 'Pipeline Status: Constructing deployment docs...', status: 'Generating release notes via local LLM...' },
    { name: 'COMPLETED', text: 'Pipeline Status: Deployment successful!', status: 'All 100% unit tests passed successfully.' }
  ];
  let currentStageIndex = 0;

  function simulatePipeline() {
    // Deactivate all nodes
    pipelineNodes.forEach(node => node.classList.remove('active'));
    
    // Get current state definition
    const state = pipelineStates[currentStageIndex];
    
    // Update badge text
    if (pipelineStateText) {
      pipelineStateText.textContent = state.text;
    }
    
    // Find active node in widget and highlight it
    const activeNode = document.querySelector(`.pipeline-node[data-stage="${state.name}"]`);
    if (activeNode) {
      activeNode.classList.add('active');
      const statusSpan = activeNode.querySelector('.node-status');
      if (statusSpan) {
        statusSpan.textContent = state.status;
      }
    }
    
    // Cycle state
    currentStageIndex = (currentStageIndex + 1) % pipelineStates.length;
  }

  // Run pipeline simulator every 4.5 seconds
  simulatePipeline();
  setInterval(simulatePipeline, 4500);

  // ─────────────────────────────────────────────
  // 5. CONTACT FORM VERIFICATION & ASYNC SUBMIT
  // ─────────────────────────────────────────────
  const contactForm = document.getElementById('contact-form');
  const overlay = document.getElementById('success-overlay');
  const overlayCloseBtn = document.getElementById('overlay-close');
  const emailInput = document.getElementById('email');
  const nameInput = document.getElementById('name');
  const messageInput = document.getElementById('message');

  // Client-side quick email validation regex
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  function validateInput(input, checkFn, errorMsg) {
    const value = input.value.trim();
    const isValid = checkFn(value);
    
    if (!isValid) {
      input.classList.add('invalid');
      let feedback = input.nextElementSibling;
      if (feedback && feedback.classList.contains('form-feedback')) {
        feedback.textContent = errorMsg;
      }
    } else {
      input.classList.remove('invalid');
    }
    return isValid;
  }

  // Real-time listener checks
  if (nameInput) {
    nameInput.addEventListener('blur', () => {
      validateInput(nameInput, val => val.length > 0, 'Name is required');
    });
  }

  if (emailInput) {
    emailInput.addEventListener('blur', () => {
      validateInput(emailInput, val => emailRegex.test(val), 'Please enter a valid email address');
    });
  }

  if (messageInput) {
    messageInput.addEventListener('blur', () => {
      validateInput(messageInput, val => val.length >= 10, 'Message must be at least 10 characters long');
    });
  }

  if (contactForm) {
    contactForm.addEventListener('submit', async (e) => {
      e.preventDefault();

      // Final validate pass
      const isNameValid = validateInput(nameInput, val => val.length > 0, 'Name is required');
      const isEmailValid = validateInput(emailInput, val => emailRegex.test(val), 'Please enter a valid email address');
      const isMsgValid = validateInput(messageInput, val => val.length >= 10, 'Message must be at least 10 characters long');

      if (!isNameValid || !isEmailValid || !isMsgValid) {
        return;
      }

      // Gather form values
      const formData = {
        name: nameInput.value.trim(),
        email: emailInput.value.trim(),
        message: messageInput.value.trim()
      };

      // Change button state
      const submitBtn = contactForm.querySelector('button[type="submit"]');
      const origBtnText = submitBtn.innerHTML;
      submitBtn.disabled = true;
      submitBtn.innerHTML = '<span class="mono">TRANSMITTING...</span>';

      try {
        const response = await fetch('/api/contact', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(formData)
        });

        const result = await response.json();

        if (response.ok && result.success) {
          // Display success state in premium overlay
          const successMsgEl = overlay.querySelector('.success-message');
          if (successMsgEl) {
            successMsgEl.textContent = result.message;
          }
          overlay.classList.add('visible');
          contactForm.reset();
        } else {
          alert(`Submission error: ${result.message || 'Server encountered an error.'}`);
        }
      } catch (err) {
        console.error('Contact transmission failed:', err);
        alert('Network transmission failed. Please verify your connection.');
      } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = origBtnText;
      }
    });
  }

  // Close overlay success panel
  if (overlayCloseBtn) {
    overlayCloseBtn.addEventListener('click', () => {
      overlay.classList.remove('visible');
    });
  }
});
