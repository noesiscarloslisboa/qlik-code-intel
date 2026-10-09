package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/cloud"
)

type cloudDependencies struct {
	httpClient *http.Client
	lookupEnv  func(string) (string, bool)
}

const cloudHelp = `Usage: qlik-repomap cloud pull --tenant HTTPS_ORIGIN --app APP_ID --out NEW_DIRECTORY [--json]

Import the latest saved script, pinned to its history version ID.
Authentication: QLIK_CLOUD_TOKEN (API key or OAuth access token).
The destination must be new, under existing directories without symlinks.
Writes exact script.qvs source and manifest.json; no reload or app changes.
Use scan/map/find/deps/context --root NEW_DIRECTORY for offline retrieval.
`

func runCloud(ctx context.Context, args []string, stdout, stderr io.Writer, deps cloudDependencies) int {
	showHelp := func(text string) int {
		if _, err := io.WriteString(stdout, text); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		return showHelp(cloudHelp)
	}
	if args[0] != "pull" {
		fmt.Fprintln(stderr, "unknown cloud command; use cloud --help")
		return 2
	}
	var tenant, appID, destination string
	var asJSON bool
	fs := flag.NewFlagSet("cloud pull", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}
	fs.StringVar(&tenant, "tenant", "", "HTTPS tenant origin")
	fs.StringVar(&appID, "app", "", "App ID whose saved script to retrieve")
	fs.StringVar(&destination, "out", "", "New snapshot directory under an existing parent")
	fs.BoolVar(&asJSON, "json", false, "Write machine-readable snapshot paths and identity")
	if err := fs.Parse(flagsFirst(fs, args[1:])); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			var text strings.Builder
			text.WriteString(cloudHelp)
			text.WriteByte('\n')
			fs.SetOutput(&text)
			fs.PrintDefaults()
			return showHelp(text.String())
		}
		return 2
	}
	if fs.NArg() != 0 || tenant == "" || appID == "" || destination == "" {
		fmt.Fprintln(stderr, "cloud pull requires --tenant, --app and --out, with no positional arguments")
		return 2
	}
	if err := cloud.ValidateTenant(tenant); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := cloud.ValidateAppID(appID); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := cloud.ValidateDestination(destination); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	token, present := deps.lookupEnv("QLIK_CLOUD_TOKEN")
	if !present || token == "" {
		fmt.Fprintln(stderr, "set QLIK_CLOUD_TOKEN to an API key or OAuth access token before cloud pull")
		return 1
	}
	client, err := cloud.NewClient(tenant, token, deps.httpClient)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	snapshot, err := client.Fetch(ctx, appID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	result, err := cloud.WriteSnapshot(ctx, destination, snapshot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(result)
	} else {
		_, err = fmt.Fprintf(stdout, "Snapshot: %s\nManifest: %s\nApp: %s\nScript: %s\n", result.SnapshotPath, result.ManifestPath, result.AppID, result.ScriptID)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
