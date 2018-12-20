package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/alecthomas/kingpin.v1"

	"github.com/toprch/syno-cli/synoapi"
)

var (
	list            = kingpin.Command("list", "List shares")
	lock            = kingpin.Command("lock", "Lock an encrypted volume")
	logout          = kingpin.Command("logout", "Logout from session")
	lockShareName   = lock.Arg("share name", "Name of the share to be locked").Required().String()
	unlock          = kingpin.Command("unlock", "Unlock an encrypted volume")
	unlockShareName = unlock.Arg("share name", "Name of the share to be unlocked").String()
	unlockBatch     = unlock.Flag("batch", "Use JSON provided via STDIN for unlocking multiple shares").Bool()
)

func main() {
	// something like https://myds.example.net:5001
	api_base := os.Getenv("SYNO_BASE_URL")
	user := os.Getenv("SYNO_USER")
	password := os.Getenv("SYNO_PASSWORD")

	command := kingpin.Parse()

	client := synoapi.NewClient(api_base)
	err := client.Login(user, password)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	default:
		kingpin.Usage()
	case "list":
		listShares(client)
	case "lock":
		lockShare(client, *lockShareName)
	case "unlock":
		unlockShare(client, *unlockShareName, *unlockBatch)
	case "logout":
		logoutShare(client)
	}
}

func listShares(client *synoapi.Client) {
	shares, err := client.ListShares()
	if err != nil {
		log.Fatal(err)
	}

	w := new(tabwriter.Writer)

	w.Init(os.Stdout, 5, 0, 2, ' ', 0)
	for _, share := range shares {
		fmt.Fprintf(w, "%s\t%s\t%s\n", share.Name, share.Encryption, share.Description)
	}
	w.Flush()
	logoutShare(client)
}

func lockShare(client *synoapi.Client, shareName string) {
	err := client.LockShare(shareName)
	if err != nil {
		log.Fatalf("Locking failed: %v", err)
	}
	logoutShare(client)

}

type ShareInfo struct {
	Name     string
	Password string
}

func unlockShare(client *synoapi.Client, shareName string, batch bool) {
	passList := make([]ShareInfo, 0)

	if batch {
		dec := json.NewDecoder(os.Stdin)
		err := dec.Decode(&passList)
		if err != nil {
			if synErr, ok := err.(*json.SyntaxError); ok {
				log.Fatalf("Failed to decode JSON: %s (offset %d)", synErr.Error(), synErr.Offset)
			} else {
				log.Fatalf("Failed to decode JSON: %v", err)
			}
			return
		}
	} else {
		if shareName == "" {
			kingpin.UsageErrorf("Please provide either a shareName or --batch. Nothing to do.")
			return
		}

		fmt.Fprint(os.Stderr, "Enter password (passing via stdin is also ok):\n")
		pass := readPassword()
		passList = append(passList, ShareInfo{shareName, pass})
	}

	for _, share := range passList {
		err := client.UnlockShare(share.Name, share.Password)

		if err != nil {
			log.Fatalf("Unlocking share '%s' failed: %v. Aborting.", share.Name, err)
			return
		}
	}
	logoutShare(client)

}

func readPassword() string {
	reader := bufio.NewReader(os.Stdin)

	// read until newline (enter pressed) and strip that newline
	pass, err := reader.ReadString('\n')
	trimmed_pass := strings.TrimRight(pass, "\n")
	if err != nil || trimmed_pass == "" {
		log.Fatalf("Failed to read password")
	}
	return trimmed_pass
}

func logoutShare(client *synoapi.Client) {
	err := client.Logout()
	if err != nil {
		log.Fatalf("Logout failed: %v", err)
	}
}
