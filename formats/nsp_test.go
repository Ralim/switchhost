package formats

import (
	"os"
	"testing"

	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
)

func TestParseNSPToMetaData(t *testing.T) {
	keyReader, err := os.Open("../testing_files/prod.keys")
	if err != nil {
		t.Fatal(err)
	}
	defer keyReader.Close()
	keystore, err := keystore.NewKeystore(keyReader)
	if err != nil {
		t.Fatal(err)
	}
	nspReader, err := os.Open("../testing_files/UnitTest_[05123A0000000000].nsp")
	if err != nil {
		t.Fatal(err)
	}
	defer nspReader.Close()
	settings := settings.NewSettings("/tmp/units.settings")
	info, err := ParseNSPToMetaData(keystore, settings, nspReader)
	if err != nil {
		t.Fatal(err)
	}
	if info.TitleID != 0x5123A0000000000 {
		t.Errorf("Should parse titleID, got 0x%X expected 0x05123A0000000000", info.TitleID)
	}
	if info.EmbeddedTitle != "UnitTest" {
		t.Errorf("Should parse embedded Title correctly, got >%s<, wanted >UnitTest<", info.EmbeddedTitle)
	}

}
