mkdir -p cmd/sftp-server
cat > cmd/sftp-server/main.go <<'EOF'
package main

import (
	"fmt"

	_ "github.com/hashicorp/vault/api"
	_ "github.com/pkg/sftp"
	_ "golang.org/x/crypto/ssh"
)

func main() {
	fmt.Println("module ok")
}
EOF
