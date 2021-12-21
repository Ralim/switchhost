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

func TestAuthCheckPasswd(t *testing.T) {

	temp_folder, err := os.MkdirTemp("", "unit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(temp_folder)
	setting := settings.NewSettings(path.Join(temp_folder, "settings.json"))
	setting.UploadingAllowed = false
	driver := NewDriver(nil, setting)
	ctx := &ftpserver.Context{
		Sess: &ftpserver.Session{
			Data: make(map[string]interface{}),
		},
	}
	//Should work if anon is on
	setting.AllowAnonFTP = false
	ok, _ := driver.CheckPasswd(ctx, "", "")

	if ok {
		t.Error("should always allow anonftp")
	}
	//test fails if anon is off and no user accounts
	setting.AllowAnonFTP = false
	ok, _ = driver.CheckPasswd(ctx, "", "")
	if ok {
		t.Error("should fail if anon is off, and no user accounts")
	}
	//Should work if user exists
	setting.Users = []settings.AuthUser{{Username: "test", Password: "testPass", AllowFTP: true, AllowUpload: true}}
	setting.AllowAnonFTP = false
	ok, _ = driver.CheckPasswd(ctx, "test", "testPass")
	if !ok {
		t.Error("should allow valid user")
	}
	if value, ok := ctx.Sess.Data["uploadAllowed"]; !ok || !value.(bool) {
		t.Error("Should set the upload status")
	}

	setting.Users = []settings.AuthUser{{Username: "test", Password: "testPass", AllowFTP: true, AllowUpload: false}}
	setting.AllowAnonFTP = false
	ok, _ = driver.CheckPasswd(ctx, "test", "testPass")
	if !ok {
		t.Error("should allow valid user")
	}
	if value, ok := ctx.Sess.Data["uploadAllowed"]; !ok || value.(bool) {
		t.Error("Should set the upload status")
	}

	setting.Users = []settings.AuthUser{{Username: "test", Password: "lololoo", AllowFTP: true, AllowUpload: true}}
	setting.AllowAnonFTP = false
	ok, _ = driver.CheckPasswd(ctx, "test", "testPass")
	if ok {
		t.Error("should block invalid user")
	}
	if value, ok := ctx.Sess.Data["uploadAllowed"]; !ok || value.(bool) {
		t.Error("Should set the upload status")
	}
	// Extra case
	// If anon ftp is allowed, but valid auth given, should handle upload perms

	setting.Users = []settings.AuthUser{{Username: "test", Password: "testPass", AllowFTP: true, AllowUpload: true}}
	setting.AllowAnonFTP = true
	ok, _ = driver.CheckPasswd(ctx, "test", "testPass")
	if !ok {
		t.Error("should allow valid user")
	}
	if value, ok := ctx.Sess.Data["uploadAllowed"]; !ok || !value.(bool) {
		t.Error("Should set the upload status")
	}

}

func TestEnsureFTPHandlesNotUsedFeatures(t *testing.T) {

	driver := NewDriver(nil, nil)
	err := driver.DeleteDir(nil, "")
	if err != ErrNotAllowed {
		t.Error("Should raise error on any delete")
	}
	err = driver.DeleteFile(nil, "")
	if err != ErrNotAllowed {
		t.Error("Should raise error on any delete")
	}
	err = driver.Rename(nil, "", "")
	if err != ErrNotAllowed {
		t.Error("Should raise error on any rename")
	}
	err = driver.MakeDir(nil, "")
	if err != nil {
		t.Error("Should silently drop MakeDir")
	}
}
