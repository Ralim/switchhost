package virtualftp

import (
	"os"
	"path"
	"testing"

	"github.com/ralim/switchhost/settings"
	ftpserver "goftp.io/server/v2"
)

func TestAuthPutFile(t *testing.T) {

	temp_folder, err := os.MkdirTemp("", "unit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(temp_folder)
	settings := settings.NewSettings(path.Join(temp_folder, "settings.json"))
	settings.UploadingAllowed = false
	driver := NewDriver(nil, settings)

	//test auth fails if no uploading allowed
	_, err = driver.PutFile(nil, "", nil, 0)
	if err != ErrNotAllowed {
		t.Error("Should disallow uploads if not permitted")
	}
	//Should fail if user is not authed correctly or not allowed upload
	ctx := &ftpserver.Context{
		Sess: &ftpserver.Session{
			Data: make(map[string]interface{}),
		},
	}
	settings.UploadingAllowed = true
	_, err = driver.PutFile(ctx, "", nil, 0)
	if err != ErrNotAllowed {
		t.Error("Should disallow uploads if not permitted")
	}
	ctx.Sess.Data["uploadAllowed"] = false
	_, err = driver.PutFile(ctx, "", nil, 0)
	if err != ErrNotAllowed {
		t.Error("Should disallow uploads if not permitted")
	}
	//Test allowed finally

	ctx.Sess.Data["uploadAllowed"] = true
	_, err = driver.PutFile(ctx, "", nil, 0)
	if err == ErrNotAllowed {
		t.Errorf("Should allow uploads if permitted - %+v", err)
	}

}
