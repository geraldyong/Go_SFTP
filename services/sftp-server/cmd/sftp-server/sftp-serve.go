package main

import (
	"errors"
	"io"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func serveSFTP(ch ssh.Channel, fs jailedFS) {
	handlers := sftp.Handlers{
		FileGet:  fs,
		FilePut:  fs,
		FileCmd:  fs,
		FileList: fs,
	}

	server := sftp.NewRequestServer(ch, handlers)

	if err := server.Serve(); err != nil && !errors.Is(err, io.EOF) {
		audit(fs.user, fs.remote, "sftp_serve_error", "", "", 0, err)
	}
	_ = server.Close()
}

