package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/CortexFoundation/CortexTheseus/accounts/keystore"
	"github.com/CortexFoundation/CortexTheseus/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var newPassphraseFlag = cli.StringFlag{
	Name:  "newpasswordfile",
	Usage: "the file that contains the new passphrase for the keyfile",
}

var commandChangePassphrase = cli.Command{
	Name:      "changepassphrase",
	Usage:     "change the passphrase on a keyfile",
	ArgsUsage: "<keyfile>",
	Description: `
Change the passphrase of a keyfile.`,
	Flags: []cli.Flag{
		passphraseFlag,
		newPassphraseFlag,
	},
	Action: func(ctx *cli.Context) error {
		keyfilepath := ctx.Args().First()

		// Read key from file.
		keyjson, err := os.ReadFile(keyfilepath)
		if err != nil {
			utils.Fatalf("Failed to read the keyfile at '%s': %v", keyfilepath, err)
		}

		// Decrypt key with passphrase.
		passphrase := getPassphrase(ctx)
		key, err := keystore.DecryptKey(keyjson, passphrase)
		if err != nil {
			utils.Fatalf("Error decrypting key: %v", err)
		}

		// Get a new passphrase.
		fmt.Println("Please provide a new passphrase")
		var newPhrase string
		if passFile := ctx.String(newPassphraseFlag.Name); passFile != "" {
			content, err := os.ReadFile(passFile)
			if err != nil {
				utils.Fatalf("Failed to read new passphrase file '%s': %v", passFile, err)
			}
			newPhrase = strings.TrimRight(string(content), "\r\n")
		} else {
			newPhrase = promptPassphrase(true)
		}

		// Encrypt the key with the new passphrase.
		newJson, err := keystore.EncryptKey(key, newPhrase, keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			utils.Fatalf("Error encrypting with new passphrase: %v", err)
		}

		// Then write the new keyfile in place of the old one.
		if err := os.WriteFile(keyfilepath, newJson, 600); err != nil {
			utils.Fatalf("Error writing new keyfile to disk: %v", err)
		}

		// Don't print anything.  Just return successfully,
		// producing a positive exit code.
		return nil
	},
}
