package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/google/go-github/v24/github"
	"github.com/nmrshll/oauth2-noserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/philips/sget/sgethash"
)

var publishReleaseSumsCmd = &cobra.Command{
	Use:   "publish-release-sums",
	Short: "Publish the release sums file for a release to a SHA256SUMS file",
	Long: `
`,
	Run: publishReleaseSumsMain,
}

func publishReleaseSumsMain(cmd *cobra.Command, args []string) {
	var releases []github.RepositoryRelease

	ctx := context.Background()

	conf := &oauth2.Config{
		ClientID:     "921edc6d2d9ca9630f89",
		ClientSecret: "1bf951fdf61abcb311baa8eecc18afc49c85ab64",
		Scopes:       []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	tc, err := oauth2ns.AuthenticateUser(conf)
	if err != nil {
		log.Fatal(err)
	}

	client := github.NewClient(tc.Client)

	owner := viper.GetString("owner")
	repo := viper.GetString("repo")
	tag := viper.GetString("tag")

	if tag != "" {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
		if err != nil {
			panic(err)
		}
		releases = append(releases, *release)
	} else {
		fmt.Printf("error: no tag and --all-releases not set")
		os.Exit(1)
	}

	urls := sgethash.URLSumList{}
	for _, r := range releases {
		for _, a := range r.Assets {
			urls.AddURL(*a.BrowserDownloadURL)
		}
		urls.AddURL(*r.ZipballURL)
		urls.AddURL(*r.TarballURL)
		sha256sumfile := urls.SHA256SumFile()

		content := []byte(sha256sumfile)
		uploadSums(client, owner, repo, tag, r, content)
	}

}

func uploadSums(client *github.Client, owner, repo, tag string, release github.RepositoryRelease, content []byte) {
	ctx := context.Background()

	tmpfile, err := ioutil.TempFile("", "sget*")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// github client, rightfully, expects to be at the beginning of the file
	tmpfile.Sync()
	tmpfile.Seek(0, 0)

	uo := github.UploadOptions{Name: "SHA256SUMS"}
	_, _, err = client.Repositories.UploadReleaseAsset(ctx, owner, repo, *release.ID, &uo, tmpfile)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
