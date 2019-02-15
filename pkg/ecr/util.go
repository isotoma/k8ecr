package ecr

import (
	"fmt"
	"os"
	"sort"

	"github.com/Masterminds/semver"
	"github.com/aws/aws-sdk-go/aws/session"
)

// LatestVersion sorts by semantic version, if there are any,
// otherwise resorts to a string sort
func latestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	vs := make([]*semver.Version, 0)
	for _, r := range versions {
		v, err := semver.NewVersion(r)
		if err == nil {
			vs = append(vs, v)
		}
	}
	if len(vs) > 0 {
		sort.Sort(semver.Collection(vs))
		return vs[len(vs)-1].Original()
	}
	sort.Strings(versions)
	return versions[len(versions)-1]
}

func createSession() *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return sess
}
