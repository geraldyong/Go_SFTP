[[ -d dev ]] || mkdir dev
[[ -f dev/sftp_host ]] || ssh-keygen -t ed25519 -f dev/sftp_host -N ""
[[ -f dev/alice ]] || ssh-keygen -t ed25519 -f dev/alice -N ""
