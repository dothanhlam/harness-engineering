#!/bin/bash
cat << 'INNER_EOF'
I have made the modifications to add a subtract function.

### FILE: random/random.go
<<<<
	return string(b)
}
====
	return string(b)
}

func Subtract(a int, b int) int {
	return a - b
}
>>>>
INNER_EOF
