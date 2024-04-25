package retest_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuluo-yx/retest"
)

func Test(t *testing.T) {
	setup()

	testInitRetestCommands_ExistEnvs(t)

	teardown()
}

func setup() {

	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
	os.Setenv("CI_DEBUG", "true")
}

func teardown() {

	os.Unsetenv("GITHUB_REPOSITORY")
	os.Unsetenv("CI_DEBUG")
}

func testInitRetestCommands_ExistEnvs(t *testing.T) {

	rt := retest.InitRetestCommands()

	assert.Equal(t, "repo", rt.Repo)
	assert.Equal(t, "owner", rt.Owner)
	assert.Equal(t, "owner/repo", rt.Nwo)
	assert.True(t, rt.Debug)

}

func testRuns(t *testing.T) {

	rt := retest.InitRetestCommands()

}
