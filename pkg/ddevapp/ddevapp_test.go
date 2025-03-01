package ddevapp_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"

	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"

	"github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	"github.com/lunixbochs/vtclean"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

var (
	DdevBin   = "ddev"
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestPkgWordpress",
			SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "wordpress-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/wordpress_files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/wordpress_db.tar.gz",
			Docroot:                       "htdocs",
			Type:                          ddevapp.AppTypeWordPress,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "this post has a photo"},
			FilesImageURI:                 "/wp-content/uploads/2017/04/pexels-photo-265186-1024x683.jpeg",
		},
		{
			Name:                          "TestPkgDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.6.1.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.6.1/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal8_6_1_files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_files.zip",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal8_6_1_db.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_db.zip",
			FullSiteTarballURL:            "",
			Type:                          ddevapp.AppTypeDrupal8,
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "this is a post with an image"},
			FilesImageURI:                 "/sites/default/files//2017-04/pexels-photo-265186.jpeg",
		},
		{
			Name:                          "TestPkgDrupal7", // Drupal Kickstart on D7
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-7.61.tar.gz",
			ArchiveInternalExtractionPath: "drupal-7.61/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.59.files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.59-db.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeDrupal7,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "D7 test project, kittens edition"},
			FilesImageURI:                 "/sites/default/files/field/image/kittens-large.jpg",
			FullSiteArchiveExtPath:        "docroot/sites/default/files",
		},
		{
			Name:                          "TestPkgDrupal6",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-6.38.tar.gz",
			ArchiveInternalExtractionPath: "drupal-6.38/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6.38_db.tar.gz",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeDrupal6,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/CHANGELOG.txt", Expect: "Drupal 6.38, 2016-02-24"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/2", Expect: "This is a story. The story is somewhat shaky"},
			FilesImageURI:                 "/sites/default/files/garland_logo.jpg",
		},
		{
			Name:                          "TestPkgBackdrop",
			SourceURL:                     "https://github.com/backdrop/backdrop/archive/1.11.0.tar.gz",
			ArchiveInternalExtractionPath: "backdrop-1.11.0/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_db.11.0.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_files.11.0.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeBackdrop,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.md", Expect: "Backdrop is a full-featured content management system"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/posts/first-post-all-about-kittens", Expect: "Lots of kittens are a good thing"},
			FilesImageURI:                 "/files/styles/large/public/field/image/kittens-large.jpg",
		},
		{
			Name:                          "TestPkgTypo3",
			SourceURL:                     "https://github.com/drud/typo3-v9-test/archive/v0.2.2.tar.gz",
			ArchiveInternalExtractionPath: "typo3-v9-test-0.2.2/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_db.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_files.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "public",
			Type:                          ddevapp.AppTypeTYPO3,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "junk readme simply for reading"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/index.php?id=65", Expect: "Boxed Content"},
			FilesImageURI:                 "/fileadmin/introduction/images/streets/nikita-maru-70928.jpg",
		},
	}
	FullTestSites = TestSites
)

func init() {
	// Make sets DDEV_BINARY_FULLPATH when building the executable
	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
}

func TestMain(m *testing.M) {
	output.LogSetUp()

	// Since this may be first time ddev has been used, we need the
	// ddev_default network available.
	dockerutil.EnsureDdevNetwork()

	// Avoid having sudo try to add to /etc/hosts.
	// This is normally done by Testsite.Prepare()
	_ = os.Setenv("DRUD_NONINTERACTIVE", "true")

	// If GOTEST_SHORT is an integer, then use it as index for a single usage
	// in the array. Any value can be used, it will default to just using the
	// first site in the array.
	gotestShort := os.Getenv("GOTEST_SHORT")
	if gotestShort != "" {
		useSite := 0
		if site, err := strconv.Atoi(gotestShort); err == nil && site >= 0 && site < len(TestSites) {
			useSite = site
		}
		TestSites = []testcommon.TestSite{TestSites[useSite]}
	}

	// testRun is the exit result we'll provide.
	// Start with a clean exit result, it will be changed if we have trouble.
	testRun := 0
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		log.Fatalf("could not read globalconfig: %v", err)
	}
	for i, site := range TestSites {

		app := &ddevapp.DdevApp{Name: site.Name}
		_ = app.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(site.Name)

		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", TestSites[i].Name, err)
		}

		switchDir := TestSites[i].Chdir()

		testcommon.ClearDockerEnv()

		app = &ddevapp.DdevApp{}
		err = app.Init(TestSites[i].Dir)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		err = app.WriteConfig()
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.WriteConfig() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		for _, volume := range []string{app.Name + "-mariadb", app.GetUnisonCatalogVolName(), app.GetWebcacheVolName()} {
			err = dockerutil.RemoveVolume(volume)
			if err != nil {
				log.Errorf("TestMain startup: Failed to delete volume %s: %v", volume, err)
			}
		}

		switchDir()
	}

	if testRun == 0 {
		log.Debugln("Running tests.")
		testRun = m.Run()
	}

	for i, site := range TestSites {
		testcommon.ClearDockerEnv()

		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		if err != nil {
			log.Fatalf("TestMain shutdown: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
		}

		if app.SiteStatus() != ddevapp.SiteStopped {
			err = app.Stop(true, false)
			if err != nil {
				log.Fatalf("TestMain shutdown: app.Stop() failed on site %s, err=%v", TestSites[i].Name, err)
			}
		}
		site.Cleanup()
	}

	os.Exit(testRun)
}

// TestDdevStart tests the functionality that is called when "ddev start" is executed
func TestDdevStart(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	// ensure docker-compose.yaml exists inside .ddev site folder
	composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
	assert.True(composeFile)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		containerName, err := constructContainerName(containerType, app)
		assert.NoError(err)
		check, err := testcommon.ContainerCheck(containerName, "running")
		assert.NoError(err)
		assert.True(check, "Container check on %s failed", containerType)
	}

	if util.IsCommandAvailable("mysql") {
		dbPort, err := app.GetPublishedPort("db")
		assert.NoError(err)

		dockerIP, _ := dockerutil.GetDockerIP()
		out, err := exec.RunCommand("mysql", []string{"--user=db", "--password=db", "--port=" + strconv.Itoa(dbPort), "--database=db", "--host=" + dockerIP, "-e", "SELECT 1;"})
		assert.NoError(err)
		assert.Contains(out, "1")
	} else {
		fmt.Print("TestDddevStart skipping check for local mysql connection because mysql command not in path")
	}

	err = app.Stop(true, false)
	assert.NoError(err)
	runTime()
	switchDir()

	// Start up TestSites[0] again with a post-start hook
	// When run the first time, it should execute the hook, second time it should not
	err = os.Chdir(site.Dir)
	assert.NoError(err)
	err = app.Init(site.Dir)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-start": {{"exec": "echo hello"}}}

	assert.NoError(err)
	stdout := util.CaptureUserOut()
	err = app.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)
	out := stdout()
	assert.Contains(out, "Running task: Exec command 'echo hello' in container/service 'web'")
	assert.Contains(out, "hello\n")

	// try to start a site of same name at different path
	another := site
	tmpDir := testcommon.CreateTmpDir("another")
	copyDir := filepath.Join(tmpDir, "copy")
	err = fileutil.CopyDir(site.Dir, copyDir)
	assert.NoError(err)
	another.Dir = copyDir
	//nolint: errcheck
	defer os.RemoveAll(copyDir)

	badapp := &ddevapp.DdevApp{}

	err = badapp.Init(copyDir)
	//nolint: errcheck
	defer badapp.Stop(true, false)
	if err == nil {
		logs, logErr := app.CaptureLogs("web", false, "")
		require.Error(t, err, "did not receive err from badapp.Init, logErr=%v, logs:\n======================= logs from app webserver =================\n%s\n============ end logs =========\n", logErr, logs)
	}
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}

	// Try to start a site of same name at an equivalent but different path. It should work.
	tmpDir, err = testcommon.OsTempDir()
	assert.NoError(err)
	symlink := filepath.Join(tmpDir, fileutil.RandomFilenameBase())
	err = os.Symlink(app.AppRoot, symlink)
	assert.NoError(err)
	//nolint: errcheck
	defer os.Remove(symlink)
	symlinkApp := &ddevapp.DdevApp{}

	err = symlinkApp.Init(symlink)
	assert.NoError(err)
	//nolint: errcheck
	defer symlinkApp.Stop(true, false)
	// Make sure that GetActiveApp() also fails when trying to start app of duplicate name in current directory.
	switchDir = another.Chdir()
	defer switchDir()

	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}
	testcommon.CleanupDir(another.Dir)
}

// TestDdevStartMultipleHostnames tests start with multiple hostnames
func TestDdevStartMultipleHostnames(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStartMultipleHostnames", site.Name))
		testcommon.ClearDockerEnv()

		err := app.Init(site.Dir)
		assert.NoError(err)

		// site.Name is explicitly added because if not removed in GetHostNames() it will cause ddev-router failure
		// "a" is repeated for the same reason; a user error of this type should not cause a failure; GetHostNames()
		// should uniqueify them.
		app.AdditionalHostnames = []string{"sub1." + site.Name, "sub2." + site.Name, "subname.sub3." + site.Name, site.Name, site.Name, site.Name}

		// sub1.<sitename>.ddev.site and sitename.ddev.site are deliberately included to prove they don't
		// cause ddev-router failures"
		app.AdditionalFQDNs = []string{"one.example.com", "two.example.com", "a.one.example.com", site.Name + "." + app.ProjectTLD, "sub1." + site.Name + "." + app.ProjectTLD}

		err = app.WriteConfig()
		assert.NoError(err)

		err = app.StartAndWaitForSync(5)
		assert.NoError(err)
		if err != nil && strings.Contains(err.Error(), "db container failed") {
			container, err := app.FindContainerByType("db")
			assert.NoError(err)
			out, err := exec.RunCommand("docker", []string{"logs", container.Names[0]})
			assert.NoError(err)
			t.Logf("DB Logs after app.Start: \n%s\n=== END DB LOGS ===", out)
		}

		// ensure docker-compose.yaml exists inside .ddev site folder
		composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "running")
			assert.NoError(err)
			assert.True(check, "Container check on %s failed", containerType)
		}

		_, _, urls := app.GetAllURLs()
		t.Logf("Testing these URLs: %v", urls)
		_, _, allURLs := app.GetAllURLs()
		for _, url := range allURLs {
			_, _ = testcommon.EnsureLocalHTTPContent(t, url+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		}

		out, err := exec.RunCommand(DdevBin, []string{"list"})
		assert.NoError(err)
		t.Logf("=========== output of ddev list ==========\n%s\n============", out)

		// Multiple projects can't run at the same time with the fqdns, so we need to clean
		// up these for tests that run later.
		app.AdditionalFQDNs = []string{}
		app.AdditionalHostnames = []string{}
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
	}
}

// TestDdevXdebugEnabled tests running with xdebug_enabled = true, etc.
func TestDdevXdebugEnabled(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevXdebugEnabled", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	// Run with xdebug_enabled: false
	testcommon.ClearDockerEnv()
	app.XdebugEnabled = false
	err = app.WriteConfig()
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, err)

	opts := &ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "php --ri xdebug",
	}
	stdout, _, err := app.Exec(opts)
	assert.Error(err)
	assert.Contains(stdout, "Extension 'xdebug' not present")

	// Run with xdebug_enabled: true
	testcommon.ClearDockerEnv()
	app.XdebugEnabled = true
	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	stdout, _, err = app.Exec(opts)
	assert.NoError(err)
	assert.Contains(stdout, "xdebug support => enabled")
	assert.Contains(stdout, "xdebug.remote_host => host.docker.internal => host.docker.internal")

	// Start a listener on port 9000 of localhost (where PHPStorm or whatever would listen)
	ln, err := net.Listen("tcp", ":9000")
	assert.NoError(err)
	// Curl to the project's index.php or anything else
	_, _, _ = testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL(), 1)

	conn, err := ln.Accept()
	assert.NoError(err)

	// Grab the Xdebug connection start and look in it for "Xdebug"
	b := make([]byte, 650)
	_, err = bufio.NewReader(conn).Read(b)
	require.NoError(t, err)
	lineString := string(b)
	assert.Contains(lineString, "Xdebug")
	runTime()
}

// TestDdevMysqlWorks tests that mysql client can be run in both containers.
func TestDdevMysqlWorks(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevMysqlWorks", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	testcommon.ClearDockerEnv()
	err = app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, err)

	// Test that mysql + .my.cnf works on web container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SELECT USER();' | grep 'db@'",
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SELECT DATABASE();' | grep 'db'",
	})
	assert.NoError(err)

	// Test that mysql + .my.cnf works on db container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'SELECT USER();' | grep 'root@localhost'",
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'SELECT DATABASE();' | grep 'db'",
	})
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()

}

// TestStartWithoutDdev makes sure we don't have a regression where lack of .ddev
// causes a panic.
func TestStartWithoutDdevConfig(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestStartWithoutDdevConfig")

	// testcommon.Chdir()() and CleanupDir() check their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	err := os.MkdirAll(testDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(testDir)
	assert.NoError(err)

	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "Could not find a project")
	}
}

// TestGetApps tests the GetActiveProjects function to ensure it accurately returns a list of running applications.
func TestGetApps(t *testing.T) {
	assert := asrt.New(t)

	// Start the apps.
	for _, site := range TestSites {
		testcommon.ClearDockerEnv()
		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)
	}

	apps := ddevapp.GetActiveProjects()

	for _, testSite := range TestSites {
		var found bool
		for _, app := range apps {
			if testSite.Name == app.GetName() {
				found = true
				break
			}
		}
		assert.True(found, "Found testSite %s in list", testSite.Name)
	}

	// Now shut down all sites as we expect them to be shut down.
	for _, site := range TestSites {
		testcommon.ClearDockerEnv()
		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)
	}
}

// TestDdevImportDB tests the functionality that is called when "ddev import-db" is executed
func TestDdevImportDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevImportDB", site.Name))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)
		err = app.StartAndWaitForSync(2)
		assert.NoError(err)

		app.Hooks = map[string][]ddevapp.YAMLTask{"post-import-db": {{"exec-host": "touch hello-post-import-db-" + app.Name}}, "pre-import-db": {{"exec-host": "touch hello-pre-import-db-" + app.Name}}}

		// Test simple db loads.
		for _, file := range []string{"users.sql", "users.mysql", "users.sql.gz", "users.mysql.gz", "users.sql.tar", "users.mysql.tar", "users.sql.tar.gz", "users.mysql.tar.gz", "users.sql.tgz", "users.mysql.tgz", "users.sql.zip", "users.mysql.zip"} {
			path := filepath.Join(testDir, "testdata", file)
			err = app.ImportDB(path, "", false)
			assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)
			if err != nil {
				continue
			}

			// Test that a settings file has correct hash_salt format
			switch app.Type {
			case ddevapp.AppTypeDrupal7:
				drupalHashSalt, err := fileutil.FgrepStringInFile(app.SiteDdevSettingsFile, "$drupal_hash_salt")
				assert.NoError(err)
				assert.True(drupalHashSalt)
			case ddevapp.AppTypeDrupal8:
				settingsHashSalt, err := fileutil.FgrepStringInFile(app.SiteDdevSettingsFile, "settings['hash_salt']")
				assert.NoError(err)
				assert.True(settingsHashSalt)
			case ddevapp.AppTypeWordPress:
				hasAuthSalt, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, "SECURE_AUTH_SALT")
				assert.NoError(err)
				assert.True(hasAuthSalt)
			}
		}

		if site.DBTarURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)
			assert.FileExists("hello-pre-import-db-" + app.Name)
			assert.FileExists("hello-post-import-db-" + app.Name)
			err = os.Remove("hello-pre-import-db-" + app.Name)
			assert.NoError(err)
			err = os.Remove("hello-post-import-db-" + app.Name)
			assert.NoError(err)

			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     "mysql -e 'SHOW TABLES;'",
			})
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))

			assert.NoError(err)
		}

		if site.DBZipURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteZipArchive", "", site.DBZipURL)

			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)

			assert.FileExists("hello-pre-import-db-" + app.Name)
			assert.FileExists("hello-post-import-db-" + app.Name)
			_ = os.RemoveAll("hello-pre-import-db-" + app.Name)
			_ = os.RemoveAll("hello-post-import-db-" + app.Name)

			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     "mysql -e 'SHOW TABLES;'",
			})
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))
		}

		if site.FullSiteTarballURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_FullSiteTarballURL", "", site.FullSiteTarballURL)
			assert.NoError(err)

			err = app.ImportDB(cachedArchive, "data.sql", false)
			assert.NoError(err, "Failed to find data.sql at root of tarball %s", cachedArchive)
			assert.FileExists("hello-pre-import-db-" + app.Name)
			assert.FileExists("hello-post-import-db-" + app.Name)
			_ = os.RemoveAll("hello-pre-import-db-" + app.Name)
			_ = os.RemoveAll("hello-post-import-db-" + app.Name)
		}
		// We don't want all the projects running at once.
		app.Hooks = nil
		_ = app.WriteConfig()
		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevOldMariaDB tests db import/export/start with Mariadb 10.1
func TestDdevOldMariaDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevOldMariaDB", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Make sure there isn't an old db laying around
	_ = dockerutil.RemoveVolume(app.Name + "-mariadb")
	//nolint: errcheck
	defer dockerutil.RemoveVolume(app.Name + "-mariadb")

	app.MariaDBVersion = ddevapp.MariaDB101
	app.DBImage = version.GetDBImage(app.MariaDBVersion)
	startErr := app.StartAndWaitForSync(15)
	//nolint: errcheck
	defer app.Stop(true, false)

	if startErr != nil {
		appLogs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync() failure %v; logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	importPath := filepath.Join(testDir, "testdata", "users.sql")
	err = app.ImportDB(importPath, "", false)
	require.NoError(t, err)

	err = os.Mkdir("tmp", 0777)
	require.NoError(t, err)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Test that we can export-db to a gzipped file
	err = app.ExportDB("tmp/users1.sql.gz", true)
	assert.NoError(err)

	// Validate contents
	err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
	assert.NoError(err)
	stringFound, err := fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Export to an ungzipped file and validate
	err = app.ExportDB("tmp/users2.sql", false)
	assert.NoError(err)

	// Validate contents
	stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Capture to stdout without gzip compression
	stdout := util.CaptureStdOut()
	err = app.ExportDB("", false)
	assert.NoError(err)
	out := stdout()
	assert.Contains(out, "Table structure for table `users`")

	snapshotName := fileutil.RandomFilenameBase()
	_, err = app.Snapshot(snapshotName)
	assert.NoError(err)
	err = app.RestoreSnapshot(snapshotName)
	assert.NoError(err)

	// Restore of a 10.2 snapshot should fail.
	// Attempt a restore with a pre-mariadb_10.2 snapshot. It should fail and give a link.
	newerSnapshotTarball, err := filepath.Abs(filepath.Join(testDir, "testdata", "restore_snapshot", "d7tester_test_1.snapshot_mariadb_10_2.tgz"))
	assert.NoError(err)

	err = archive.Untar(newerSnapshotTarball, filepath.Join(site.Dir, ".ddev", "db_snapshots"), "")
	assert.NoError(err)
	err = app.RestoreSnapshot("d7testerTest1")
	assert.Error(err)
	assert.Contains(err.Error(), "is not compatible")

	runTime()
}

// TestDdevExportDB tests the functionality that is called when "ddev export-db" is executed
func TestDdevExportDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExportDB", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)
	importPath := filepath.Join(testDir, "testdata", "users.sql")
	err = app.ImportDB(importPath, "", false)
	require.NoError(t, err)

	_ = os.Mkdir("tmp", 0777)
	// Most likely reason for failure is it exists, so let that go
	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Test that we can export-db to a gzipped file
	err = app.ExportDB("tmp/users1.sql.gz", true)
	assert.NoError(err)

	// Validate contents
	err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
	assert.NoError(err)
	stringFound, err := fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Export to an ungzipped file and validate
	err = app.ExportDB("tmp/users2.sql", false)
	assert.NoError(err)

	// Validate contents
	stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Capture to stdout without gzip compression
	stdout := util.CaptureStdOut()
	err = app.ExportDB("", false)
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "Table structure for table `users`")

	// Try it with capture to stdout, validate contents.
	runTime()
}

// TestDdevFullSiteSetup tests a full import-db and import-files and then looks to see if
// we have a spot-test success hit on a URL
func TestDdevFullSiteSetup(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		defer switchDir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevFullSiteSetup", site.Name))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// Get files before start, as syncing can start immediately.
		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		// Running WriteConfig assures that settings.ddev.php gets written
		// so Drupal 8 won't try to set things unwriteable
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)
		settingsLocation, err := app.DetermineSettingsPathLocation()
		assert.NoError(err)
		assert.Equal(filepath.Dir(settingsLocation), filepath.Dir(app.SiteSettingsPath))
		if nodeps.ArrayContainsString([]string{"drupal6", "drupal7", "drupal8", "backdrop"}, app.Type) {
			assert.FileExists(filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
		}
		if app.Type == "drupal8" {
			assert.FileExists(filepath.Join(filepath.Dir(app.SiteSettingsPath), "..", "all", "drush", "drush.yml"))
		}

		if site.DBTarURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)
		}

		startErr := app.StartAndWaitForSync(2)
		if startErr != nil {
			appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			t.Fatalf("app.StartAndWaitForSync() failure err=%v; logs:\n=====\n%s\n=====\n", startErr, appLogs)
		}

		// Test static content.
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		// Test dynamic php + database content.
		rawurl := app.GetHTTPURL() + site.DynamicURI.URI
		body, resp, err := testcommon.GetLocalHTTPResponse(t, rawurl, 60)
		assert.NoError(err, "GetLocalHTTPResponse returned err on project=%s rawurl %s, resp=%v: %v", site.Name, rawurl, resp, err)
		if err != nil && strings.Contains(err.Error(), "container ") {
			logs, err := ddevapp.GetErrLogsFromApp(app, err)
			assert.NoError(err)
			t.Fatalf("Logs after GetLocalHTTPResponse: %s", logs)
		}
		assert.Contains(body, site.DynamicURI.Expect, "expected %s on project %s", site.DynamicURI.Expect, site.Name)

		// Load an image from the files section
		if site.FilesImageURI != "" {
			_, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL()+site.FilesImageURI)
			assert.NoError(err, "failed ImageURI response on project %s: %v", site.Name, err)
			assert.Equal("image/jpeg", resp.Header["Content-Type"][0])
		}

		// Make sure we can do a simple hit against the host-mount of web container.
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetWebContainerDirectHTTPURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

		// We don't want all the projects running at once.
		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevRestoreSnapshot tests creating a snapshot and reverting to it. This runs with Mariadb 10.2
func TestDdevRestoreSnapshot(t *testing.T) {
	assert := asrt.New(t)
	testDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("TestDdevRestoreSnapshot"))

	d7testerTest1Dump, err := filepath.Abs(filepath.Join("testdata", "restore_snapshot", "d7tester_test_1.sql.gz"))
	assert.NoError(err)
	d7testerTest2Dump, err := filepath.Abs(filepath.Join("testdata", "restore_snapshot", "d7tester_test_2.sql.gz"))
	assert.NoError(err)

	// Use d7 only for this test, the key thing is the database interaction
	site := FullTestSites[2]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		err = site.Prepare()
		require.NoError(t, err)
	}

	switchDir := site.Chdir()
	defer switchDir()

	testcommon.ClearDockerEnv()

	err = app.Init(site.Dir)
	require.NoError(t, err)

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-snapshot": {{"exec-host": "touch hello-post-snapshot-" + app.Name}}, "pre-snapshot": {{"exec-host": "touch hello-pre-snapshot-" + app.Name}}}

	// Try using php72 to avoid SIGBUS failures after restore.
	app.PHPVersion = ddevapp.PHP72

	// First do regular start, which is good enough to get us to an ImportDB()
	err = app.Start()
	require.NoError(t, err)

	err = app.ImportDB(d7testerTest1Dump, "", false)
	require.NoError(t, err, "Failed to app.ImportDB path: %s err: %v", d7testerTest1Dump, err)

	err = app.StartAndWaitForSync(2)
	require.NoError(t, err, "app.Start() failed on site %s, err=%v", site.Name, err)

	resp, ensureErr := testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node", 45)
	assert.NoError(ensureErr)
	if ensureErr != nil && strings.Contains(ensureErr.Error(), "container failed") {
		logs, err := ddevapp.GetErrLogsFromApp(app, ensureErr)
		assert.NoError(err)
		t.Fatalf("container failed: logs:\n=======\n%s\n========\n", logs)
	}
	require.NotNil(t, resp)
	if ensureErr != nil && resp.StatusCode != 200 {
		logs, err := app.CaptureLogs("web", false, "")
		assert.NoError(err)
		t.Fatalf("EnsureLocalHTTPContent received %d. Resp=%v, web logs=\n========\n%s\n=========\n", resp.StatusCode, resp, logs)
	}

	// Make a snapshot of d7 tester test 1
	backupsDir := filepath.Join(app.GetConfigPath(""), "db_snapshots")
	snapshotName, err := app.Snapshot("d7testerTest1")
	assert.NoError(err)

	assert.EqualValues(snapshotName, "d7testerTest1")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	assert.FileExists("hello-pre-snapshot-" + app.Name)
	assert.FileExists("hello-post-snapshot-" + app.Name)
	err = os.Remove("hello-pre-snapshot-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-snapshot-" + app.Name)
	assert.NoError(err)

	err = app.ImportDB(d7testerTest2Dump, "", false)
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", d7testerTest2Dump, err)
	_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 2 has 2 nodes", 45)

	snapshotName, err = app.Snapshot("d7testerTest2")
	assert.NoError(err)
	assert.EqualValues(snapshotName, "d7testerTest2")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-restore-snapshot": {{"exec-host": "touch hello-post-restore-snapshot-" + app.Name}}, "pre-restore-snapshot": {{"exec-host": "touch hello-pre-restore-snapshot-" + app.Name}}}

	err = app.RestoreSnapshot("d7testerTest1")
	assert.NoError(err)

	assert.FileExists("hello-pre-restore-snapshot-" + app.Name)
	assert.FileExists("hello-post-restore-snapshot-" + app.Name)
	err = os.Remove("hello-pre-restore-snapshot-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-restore-snapshot-" + app.Name)
	assert.NoError(err)

	_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node", 45)
	err = app.RestoreSnapshot("d7testerTest2")
	assert.NoError(err)

	body, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL(), 45)
	assert.NoError(err, "GetLocalHTTPResponse returned err on rawurl %s: %v", app.GetHTTPURL(), err)
	assert.Contains(body, "d7 tester test 2 has 2 nodes")
	if err != nil {
		t.Logf("resp after timeout: %v", resp)
		out, err := app.CaptureLogs("web", false, "")
		assert.NoError(err)
		t.Logf("web container logs after timeout: %s", out)
	}

	// Attempt a restore with a pre-mariadb_10.2 snapshot. It should fail and give a link.
	oldSnapshotTarball, err := filepath.Abs(filepath.Join(testDir, "testdata", "restore_snapshot", "d7tester_test_1.snapshot_mariadb_10_1.tgz"))
	assert.NoError(err)

	err = archive.Untar(oldSnapshotTarball, filepath.Join(site.Dir, ".ddev", "db_snapshots", "oldsnapshot"), "")
	assert.NoError(err)
	err = app.RestoreSnapshot("oldsnapshot")
	assert.Error(err)
	assert.Contains(err.Error(), "is not compatible")

	app.Hooks = nil
	_ = app.WriteConfig()
	err = app.Stop(true, false)
	assert.NoError(err)

	// TODO: Check behavior of ddev rm with snapshot, see if it has right stuff in it.

	runTime()
}

// TestWriteableFilesDirectory tests to make sure that files created on host are writable on container
// and files ceated in container are correct user on host.
func TestWriteableFilesDirectory(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestWritableFilesDirectory", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.StartAndWaitForSync(0)
	assert.NoError(err)

	uploadDir := app.GetUploadDir()
	assert.NotEmpty(uploadDir)

	// Use exec to touch a file in the container and see what the result is. Make sure it comes out with ownership
	// making it writeable on the host.
	filename := fileutil.RandomFilenameBase()
	dirname := fileutil.RandomFilenameBase()
	// Use path.Join for items on th container (linux) and filepath.Join for items on the host.
	inContainerDir := path.Join(uploadDir, dirname)
	onHostDir := filepath.Join(app.Docroot, inContainerDir)

	// The container execution directory is dependent on the app type
	switch app.Type {
	case ddevapp.AppTypeWordPress, ddevapp.AppTypeTYPO3, ddevapp.AppTypePHP:
		inContainerDir = path.Join(app.Docroot, inContainerDir)
	}

	inContainerRelativePath := path.Join(inContainerDir, filename)
	onHostRelativePath := path.Join(onHostDir, filename)

	err = os.MkdirAll(onHostDir, 0775)
	assert.NoError(err)
	// Create a file in the directory to make sure it syncs
	f, err := os.OpenFile(filepath.Join(onHostDir, "junk.txt"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	assert.NoError(err)
	_ = f.Close()
	ddevapp.WaitForSync(app, 5)

	_, _, createFileErr := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "echo 'content created inside container\n' >" + inContainerRelativePath,
	})
	assert.NoError(createFileErr)
	if app.WebcacheEnabled && createFileErr != nil {
		syncLogs, err := app.CaptureLogs("bgsync", false, "")
		assert.NoError(err)

		// ls -lR on host
		onHostList, err := exec.RunCommand("ls", []string{"-lR", filepath.Dir(uploadDir)})
		assert.NoError(err)

		// ls -lR in container
		inContainerList, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     "ls -lR " + filepath.Dir(uploadDir),
		})
		assert.NoError(err)
		t.Fatalf("Unable to create file %s inside container; onHost ls=\n====\n%s\n====\ninContainer ls=\n======\n%s\n=====\nContainer Sync logs=\n=======\n%s\n===========\n", inContainerRelativePath, onHostList, inContainerList, syncLogs)
	}

	ddevapp.WaitForSync(app, 5)

	// Now try to append to the file on the host.
	// os.OpenFile() for append here fails if the file does not already exist.
	f, err = os.OpenFile(onHostRelativePath, os.O_APPEND|os.O_WRONLY, 0660)
	assert.NoError(err)
	_, err = f.WriteString("this addition to the file was added on the host side")
	assert.NoError(err)
	_ = f.Close()

	// Create a file on the host and see what the result is. Make sure we can not append/write to it in the container.
	filename = fileutil.RandomFilenameBase()
	dirname = fileutil.RandomFilenameBase()
	inContainerDir = path.Join(uploadDir, dirname)
	onHostDir = filepath.Join(app.Docroot, inContainerDir)
	// The container execution directory is dependent on the app type
	switch app.Type {
	case ddevapp.AppTypeWordPress, ddevapp.AppTypeTYPO3, ddevapp.AppTypePHP:
		inContainerDir = path.Join(app.Docroot, inContainerDir)
	}

	inContainerRelativePath = path.Join(inContainerDir, filename)
	onHostRelativePath = filepath.Join(onHostDir, filename)

	err = os.MkdirAll(onHostDir, 0775)
	assert.NoError(err)

	f, err = os.OpenFile(onHostRelativePath, os.O_CREATE|os.O_RDWR, 0660)
	assert.NoError(err)
	_, err = f.WriteString("this base content was inserted on the host side\n")
	assert.NoError(err)
	_ = f.Close()

	ddevapp.WaitForSync(app, 5)

	// if the file exists, add to it. We don't want to add if it's not already there.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "if [ -f " + inContainerRelativePath + " ]; then echo 'content added inside container\n' >>" + inContainerRelativePath + "; fi",
	})
	assert.NoError(err)
	// grep the file for both the content added on host and that added in container.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "grep 'base content was inserted on the host' " + inContainerRelativePath + "&& grep 'content added inside container' " + inContainerRelativePath,
	})
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestDdevImportFilesDir tests that "ddev import-files" can successfully import non-archive directories
func TestDdevImportFilesDir(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Create a dummy directory to test non-archive imports
	importDir, err := ioutil.TempDir("", t.Name())
	assert.NoError(err)
	fileNames := make([]string, 0)
	for i := 0; i < 5; i++ {
		fileName := uuid.New().String()
		fileNames = append(fileNames, fileName)

		fullPath := filepath.Join(importDir, fileName)
		err = ioutil.WriteFile(fullPath, []byte(fileName), 0644)
		assert.NoError(err)
	}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		// Function under test
		err = app.ImportFiles(importDir, "")
		assert.NoError(err, "Importing a directory returned an error:", err)

		// Confirm contents of destination dir after import
		absUploadDir := filepath.Join(app.AppRoot, app.Docroot, app.GetUploadDir())
		uploadedFiles, err := ioutil.ReadDir(absUploadDir)
		assert.NoError(err)

		uploadedFilesMap := map[string]bool{}
		for _, uploadedFile := range uploadedFiles {
			uploadedFilesMap[filepath.Base(uploadedFile.Name())] = true
		}

		for _, expectedFile := range fileNames {
			assert.True(uploadedFilesMap[expectedFile], "Expected file %s not found for site: %s", expectedFile, site.Name)
		}

		runTime()
		switchDir()
	}
}

// TestDdevImportFiles tests the functionality that is called when "ddev import-files" is executed
func TestDdevImportFiles(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)
		app.Hooks = map[string][]ddevapp.YAMLTask{"post-import-files": {{"exec-host": "touch hello-post-import-files-" + app.Name}}, "pre-import-files": {{"exec-host": "touch hello-pre-import-files-" + app.Name}}}

		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(zipballPath, "")
			assert.NoError(err)
		}

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			assert.NoError(err)
		}
		assert.FileExists("hello-pre-import-files-" + app.Name)
		assert.FileExists("hello-post-import-files-" + app.Name)
		err = os.Remove("hello-pre-import-files-" + app.Name)
		assert.NoError(err)
		err = os.Remove("hello-post-import-files-" + app.Name)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevImportFilesCustomUploadDir ensures that files are imported to a custom upload directory when requested
func TestDdevImportFilesCustomUploadDir(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// Set custom upload dir
		app.UploadDir = "my/upload/dir"
		absUploadDir := filepath.Join(app.AppRoot, app.Docroot, app.UploadDir)
		err = os.MkdirAll(absUploadDir, 0755)
		assert.NoError(err)

		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(zipballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		runTime()
		switchDir()
	}
}

// TestDdevExec tests the execution of commands inside a docker container of a site.
func TestDdevExec(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExec", site.Name))

		err := app.Init(site.Dir)
		assert.NoError(err)

		app.Hooks = map[string][]ddevapp.YAMLTask{"post-exec": {{"exec-host": "touch hello-post-exec-" + app.Name}}, "pre-exec": {{"exec-host": "touch hello-pre-exec-" + app.Name}}}
		defer func() {
			app.Hooks = nil
			_ = app.WriteConfig()
		}()

		startErr := app.StartAndWaitForSync(0)
		if startErr != nil {
			logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(err)
			t.Fatalf("app.StartAndWaitForSync() failed err=%v, logs from broken container:\n=======\n%s\n========\n", startErr, logs)
		}

		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     "pwd",
		})
		assert.NoError(err)
		assert.Contains(out, "/var/www/html")

		assert.FileExists("hello-pre-exec-" + app.Name)
		assert.FileExists("hello-post-exec-" + app.Name)
		err = os.Remove("hello-pre-exec-" + app.Name)
		assert.NoError(err)
		err = os.Remove("hello-post-exec-" + app.Name)
		assert.NoError(err)

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Dir:     "/usr/local",
			Cmd:     "pwd",
		})
		assert.NoError(err)
		assert.Contains(out, "/usr/local")

		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     "mysql -e 'DROP DATABASE db;'",
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     "mysql information_schema -e 'CREATE DATABASE db;'",
		})
		assert.NoError(err)

		switch app.GetType() {
		case ddevapp.AppTypeDrupal6:
			fallthrough
		case ddevapp.AppTypeDrupal7:
			fallthrough
		case ddevapp.AppTypeDrupal8:
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "drush status",
			})
			assert.NoError(err)
			assert.Regexp("PHP configuration[ :]*/etc/php/[0-9].[0-9]/cli/php.ini", out)
		case ddevapp.AppTypeWordPress:
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "wp --info",
			})
			assert.NoError(err)
			assert.Regexp("/etc/php.*/php.ini", out)
		}

		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevLogs tests the container log output functionality.
func TestDdevLogs(t *testing.T) {
	assert := asrt.New(t)

	// Skip test because on Windows because the CaptureUserOut() hangs, at least
	// sometimes.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test TestDdevLogs on Windows")
	}

	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevLogs", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	//nolint: errcheck
	defer app.Stop(true, false)

	startErr := app.StartAndWaitForSync(0)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.Start failed, err=%v, logs=\n========\n%s\n===========\n", startErr, logs)
	}

	out, err := app.CaptureLogs("web", false, "")
	assert.NoError(err)
	assert.Contains(out, "Server started")

	out, err = app.CaptureLogs("db", false, "")
	assert.NoError(err)
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	// Test that we can get logs when project is stopped also
	err = app.Pause()
	assert.NoError(err)

	out, err = app.CaptureLogs("web", false, "")
	assert.NoError(err)
	assert.Contains(out, "Server started")

	out, err = app.CaptureLogs("db", false, "")
	assert.NoError(err)
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	runTime()
	switchDir()
}

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	assert := asrt.New(t)

	// Use Drupal8 only, mostly for the composer example
	site := FullTestSites[1]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		app := &ddevapp.DdevApp{Name: site.Name}
		_ = app.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(site.Name)

		err := site.Prepare()
		require.NoError(t, err)
		// nolint: errcheck
		defer os.RemoveAll(site.Dir)
	}

	cleanup := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s ProcessHooks", site.Name))

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true, ddevapp.ProviderDefault)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
	assert.NoError(err)

	// Note that any ExecHost commands must be able to run on Windows.
	// echo and pwd are things that work pretty much the same in both places.
	app.Hooks = map[string][]ddevapp.YAMLTask{
		"hook-test": {
			{"exec": "ls /usr/local/bin/composer"},
			{"exec-host": "echo something"},
			{"exec": "echo MYSQL_USER=${MYSQL_USER}", "service": "db"},
			{"exec": "echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt"},
			{"exec": "touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt"},
			{"composer": "show twig/twig"},
		},
	}

	stdout := util.CaptureUserOut()
	err = app.ProcessHooks("hook-test")
	assert.NoError(err)

	// Ignore color in output, can be different in different OS's
	out := vtclean.Clean(stdout(), false)

	assert.Contains(out, "Executing hook-test hook")
	assert.Contains(out, "Exec command 'ls /usr/local/bin/composer' in container/service 'web'")
	assert.Contains(out, "Exec command 'echo something' on the host")
	assert.Contains(out, "Exec command 'echo MYSQL_USER=${MYSQL_USER}' in container/service 'db'")
	assert.Contains(out, "MYSQL_USER=db")
	assert.Contains(out, "Exec command 'echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt' in container/service 'web'")
	assert.Contains(out, "Exec command 'touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt' in container/service 'web',")
	assert.Contains(out, "Twig, the flexible, fast, and secure template")
	assert.FileExists(filepath.Join(app.AppRoot, fmt.Sprintf("TestProcessHooks%s.txt", app.RouterHTTPSPort)))
	assert.FileExists(filepath.Join(app.AppRoot, "touch_works_after_and.txt"))

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
	cleanup()
}

// TestDdevPause tests the functionality that is called when "ddev pause" is executed
func TestDdevPause(t *testing.T) {
	assert := asrt.New(t)

	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStop", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-pause": {{"exec-host": "touch hello-post-pause-" + app.Name}}, "pre-pause": {{"exec-host": "touch hello-pre-pause-" + app.Name}}}

	defer func() {
		app.Hooks = nil
		_ = app.WriteConfig()
		_ = app.Stop(true, false)
	}()
	require.NoError(t, err)
	err = app.Pause()
	assert.NoError(err)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		containerName, err := constructContainerName(containerType, app)
		assert.NoError(err)
		check, err := testcommon.ContainerCheck(containerName, "exited")
		assert.NoError(err)
		assert.True(check, containerType, "container has exited")
	}
	pwd, _ := os.Getwd()
	_ = pwd
	assert.FileExists("hello-pre-pause-" + app.Name)
	assert.FileExists("hello-post-pause-" + app.Name)
	err = os.Remove("hello-pre-pause-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-pause-" + app.Name)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestDdevStopMissingDirectory tests that the 'ddev stop' command works properly on sites with missing directories or ddev configs.
func TestDdevStopMissingDirectory(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	testcommon.ClearDockerEnv()
	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)

	startErr := app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}

	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	// Move the site directory to a temp location to mimic a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	// ddev stop (in cmd) actually does the check for missing project files,
	// so we imitate that here.
	err = ddevapp.CheckForMissingProjectFiles(app)
	assert.Error(err)
	assert.Contains(err.Error(), "If you would like to continue using ddev to manage this project please restore your files to that directory.")
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
}

// TestDdevDescribe tests that the describe command works properly on a running
// and also a stopped project.
func TestDdevDescribe(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-describe": {{"exec-host": "touch hello-post-describe-" + app.Name}}, "pre-describe": {{"exec-host": "touch hello-pre-describe-" + app.Name}}}

	startErr := app.StartAndWaitForSync(0)
	defer func() {
		_ = app.Stop(true, false)
		app.Hooks = nil
		_ = app.WriteConfig()
	}()
	// If we have a problem starting, get the container logs and output.
	if startErr != nil {
		out, logsErr := app.CaptureLogs("web", false, "")
		assert.NoError(logsErr)

		healthcheck, inspectErr := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect ddev-%s-web|jq -r '.[0].State.Health.Log[-1]'", app.Name)})
		assert.NoError(inspectErr)

		t.Fatalf("app.StartAndWaitForSync(%s) failed: %v, \nweb container healthcheck='%s', \n=== web container logs=\n%s\n=== END web container logs ===", site.Name, err, healthcheck, out)
	}

	desc, err := app.Describe()
	assert.NoError(err)
	assert.EqualValues(ddevapp.SiteRunning, desc["status"], "")
	assert.EqualValues(app.GetName(), desc["name"])
	assert.EqualValues(ddevapp.RenderHomeRootedDir(app.GetAppRoot()), desc["shortroot"])
	assert.EqualValues(app.GetAppRoot(), desc["approot"])
	assert.EqualValues(app.GetPhpVersion(), desc["php_version"])

	assert.FileExists("hello-pre-describe-" + app.Name)
	assert.FileExists("hello-post-describe-" + app.Name)
	err = os.Remove("hello-pre-describe-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-describe-" + app.Name)
	assert.NoError(err)

	// Now stop it and test behavior.
	err = app.Pause()
	assert.NoError(err)

	desc, err = app.Describe()
	assert.NoError(err)
	assert.EqualValues(ddevapp.SitePaused, desc["status"])

	switchDir()
}

// TestDdevDescribeMissingDirectory tests that the describe command works properly on sites with missing directories or ddev configs.
func TestDdevDescribeMissingDirectory(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}
	// Move the site directory to a temp location to mimick a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	desc, err := app.Describe()
	assert.NoError(err)
	assert.Contains(desc["status"], ddevapp.SiteDirMissing, "Status did not include the phrase '%s' when describing a site with missing directories.", ddevapp.SiteDirMissing)
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
}

// TestRouterPortsCheck makes sure that we can detect if the ports are available before starting the router.
func TestRouterPortsCheck(t *testing.T) {
	assert := asrt.New(t)

	// First, stop any sites that might be running
	app := &ddevapp.DdevApp{}

	// Stop/Remove all sites, which should get the router out of there.
	for _, site := range TestSites {
		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		if app.SiteStatus() == ddevapp.SiteRunning || app.SiteStatus() == ddevapp.SitePaused {
			err = app.Stop(true, false)
			assert.NoError(err)
		}

		switchDir()
	}

	// Now start one site, it's hard to get router to behave without one site.
	site := TestSites[0]
	testcommon.ClearDockerEnv()

	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWaitForSync(5)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync() failure; err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	app, err = ddevapp.GetActiveApp(site.Name)
	if err != nil {
		t.Fatalf("Failed to GetActiveApp(%s), err:%v", site.Name, err)
	}
	startErr = app.StartAndWaitForSync(5)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync() failure err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	// Stop the router using code from StopRouterIfNoContainers().
	// StopRouterIfNoContainers can't be used here because it checks to see if containers are running
	// and doesn't do its job as a result.
	dest := ddevapp.RouterComposeYAMLPath()
	_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down")
	assert.NoError(err, "Failed to stop router using docker-compose, err=%v", err)

	// Occupy port 80 using docker busybox trick, then see if we can start router.
	// This is done with docker so that we don't have to use explicit sudo
	containerID, err := exec.RunCommand("sh", []string{"-c", "docker run -d -p80:80 --rm busybox:latest sleep 100 2>/dev/null"})
	if err != nil {
		t.Fatalf("Failed to run docker command to occupy port 80, err=%v output=%v", err, containerID)
	}
	containerID = strings.TrimSpace(containerID)

	// Now try to start the router. It should fail because the port is occupied.
	err = ddevapp.StartDdevRouter()
	assert.Error(err, "Failure: router started even though port 80 was occupied")

	// Remove our dummy busybox docker container.
	out, err := exec.RunCommand("docker", []string{"rm", "-f", containerID})
	assert.NoError(err, "Failed to docker rm the port-occupier container, err=%v output=%v", err, out)
}

// TestCleanupWithoutCompose ensures app containers can be properly cleaned up without a docker-compose config file present.
func TestCleanupWithoutCompose(t *testing.T) {
	assert := asrt.New(t)

	// Skip test because we can't rename folders while they're in use if running on Windows.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test TestCleanupWithoutCompose on Windows")
	}

	site := TestSites[0]

	revertDir := site.Chdir()
	app := &ddevapp.DdevApp{}

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Ensure we have a site started so we have something to cleanup

	startErr := app.StartAndWaitForSync(5)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync failure; err=%v, logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}
	// Setup by creating temp directory and nesting a folder for our site.
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")

	//nolint: errcheck
	defer os.RemoveAll(tempPath)
	//nolint: errcheck
	defer revertDir()
	// Move the site directory back to its original location.
	//nolint: errcheck
	defer os.Rename(siteCopyDest, site.Dir)

	// Move site directory to a temp directory to mimick a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	// Call the Stop command()
	// Notice that we set the removeData parameter to true.
	// This gives us added test coverage over sites with missing directories
	// by ensuring any associated database files get cleaned up as well.
	err = app.Stop(true, false)
	assert.NoError(err)
	assert.Empty(globalconfig.DdevGlobalConfig.ProjectList[app.Name])

	for _, containerType := range [3]string{"web", "db", "dba"} {
		_, err := constructContainerName(containerType, app)
		assert.Error(err)
	}

	// Ensure there are no volumes associated with this project
	client := dockerutil.GetDockerClient()
	volumes, err := client.ListVolumes(docker.ListVolumesOptions{})
	assert.NoError(err)
	for _, volume := range volumes {
		assert.False(volume.Labels["com.docker.compose.project"] == "ddev"+strings.ToLower(app.GetName()))
	}

}

// TestGetappsEmpty ensures that GetActiveProjects returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := asrt.New(t)

	// Ensure test sites are removed
	for _, site := range TestSites {
		app := &ddevapp.DdevApp{}

		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		if app.SiteStatus() != ddevapp.SiteStopped {
			err = app.Stop(true, false)
			assert.NoError(err)
		}
		switchDir()
	}

	apps := ddevapp.GetActiveProjects()
	assert.Equal(0, len(apps), "Expected to find no apps but found %d apps=%v", len(apps), apps)
}

// TestRouterNotRunning ensures the router is shut down after all sites are stopped.
// This depends on TestGetAppsEmpty() having shut everything down.
func TestRouterNotRunning(t *testing.T) {
	assert := asrt.New(t)
	containers, err := dockerutil.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual("ddev-router", dockerutil.ContainerName(container), "ddev-router was not supposed to be running but it was")
	}
}

// TestListWithoutDir prevents regression where ddev list panics if one of the
// sites found is missing a directory
func TestListWithoutDir(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	packageDir, _ := os.Getwd()

	// startCount is the count of apps at the start of this adventure
	apps := ddevapp.GetActiveProjects()
	startCount := len(apps)

	testDir := testcommon.CreateTmpDir("TestStartWithoutDdevConfig")
	defer testcommon.CleanupDir(testDir)

	err := os.MkdirAll(testDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, true, ddevapp.ProviderDefault)
	assert.NoError(err)
	app.Name = "junk"
	app.Type = ddevapp.AppTypeDrupal7
	err = app.WriteConfig()
	assert.NoError(err)

	// Do a start on the configured site.
	app, err = ddevapp.GetActiveApp("")
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	// Make sure we move out of the directory for Windows' sake
	garbageDir := testcommon.CreateTmpDir("RestingHere")
	defer testcommon.CleanupDir(garbageDir)

	err = os.Chdir(garbageDir)
	assert.NoError(err)

	testcommon.CleanupDir(testDir)

	apps = ddevapp.GetActiveProjects()

	assert.EqualValues(len(apps), startCount+1)

	// Make a whole table and make sure our app directory missing shows up.
	// This could be done otherwise, but we'd have to go find the site in the
	// array first.
	table := ddevapp.CreateAppTable()
	for _, site := range apps {
		desc, err := site.Describe()
		if err != nil {
			t.Fatalf("Failed to describe site %s: %v", site.GetName(), err)
		}

		ddevapp.RenderAppRow(table, desc)
	}

	// testDir on Windows has backslashes in it, resulting in invalid regexp
	// Remove them and use ., which is good enough.
	testDirSafe := strings.Replace(testDir, "\\", ".", -1)
	assert.Regexp(regexp.MustCompile("(?s)"+ddevapp.SiteDirMissing+".*"+testDirSafe), table.String())

	err = app.Stop(true, false)
	assert.NoError(err)

	// Change back to package dir. Lots of things will have to be cleaned up
	// in defers, and for windows we have to not be sitting in them.
	err = os.Chdir(packageDir)
	assert.NoError(err)
}

type URLRedirectExpectations struct {
	scheme              string
	uri                 string
	expectedRedirectURI string
}

// TestHttpsRedirection tests to make sure that webserver and php redirect to correct
// scheme (http or https).
func TestHttpsRedirection(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	packageDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir("TestHttpsRedirection")
	defer testcommon.CleanupDir(testDir)
	appDir := filepath.Join(testDir, "proj")
	err := fileutil.CopyDir(filepath.Join(packageDir, "testdata", "TestHttpsRedirection"), appDir)
	assert.NoError(err)
	err = os.Chdir(appDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(appDir, true, ddevapp.ProviderDefault)
	assert.NoError(err)
	app.Name = "proj"
	app.Type = ddevapp.AppTypePHP

	expectations := []URLRedirectExpectations{
		{"https", "/subdir", "/subdir/"},
		{"https", "/redir_abs.php", "/landed.php"},
		{"https", "/redir_relative.php", "/landed.php"},
		{"http", "/subdir", "/subdir/"},
		{"http", "/redir_abs.php", "/landed.php"},
		{"http", "/redir_relative.php", "/landed.php"},
	}

	for _, webserverType := range []string{ddevapp.WebserverNginxFPM, ddevapp.WebserverApacheFPM, ddevapp.WebserverApacheCGI} {
		app.WebserverType = webserverType
		err = app.WriteConfig()
		assert.NoError(err)

		// Do a start on the configured site.
		app, err = ddevapp.GetActiveApp("")
		assert.NoError(err)
		// Make absolutely sure the project is stopped
		err = app.Stop(true, false)
		assert.NoError(err)
		//nolint: errcheck
		defer app.Stop(true, false)
		startErr := app.StartAndWaitForSync(30)
		if startErr != nil {
			appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			// Get healthcheck status on bgsync container
			healthcheck, inspectErr := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect ddev-%s-bgsync|jq -r '.[0].State.Health.Log[-1]'", app.Name)})
			assert.NoError(inspectErr)

			t.Fatalf("app.StartAndWaitForSync failure; err=%v \n===== container logs ===\n%s\n===== bgsync health info ===\n%s\n========\n", startErr, appLogs, healthcheck)
		}
		// Test for directory redirects under https and http
		for _, parts := range expectations {

			reqURL := parts.scheme + "://" + app.GetHostname() + parts.uri
			t.Logf("TestHttpsRedirection trying URL %s with webserver_type=%s", reqURL, webserverType)
			out, resp, err := testcommon.GetLocalHTTPResponse(t, reqURL)
			assert.NotNil(resp, "resp was nil for webserver_type=%s url=%s, err=%v, out='%s'", webserverType, reqURL, err, out)
			if resp != nil {
				locHeader := resp.Header.Get("Location")

				expectedRedirect := parts.expectedRedirectURI
				// However, if we're hitting redir_abs.php (or apache hitting directory), the redirect will be the whole url.
				if strings.Contains(parts.uri, "redir_abs.php") || webserverType != ddevapp.WebserverNginxFPM {
					expectedRedirect = parts.scheme + "://" + app.GetHostname() + parts.expectedRedirectURI
				}
				// Except the php relative redirect is always relative.
				if strings.Contains(parts.uri, "redir_relative.php") {
					expectedRedirect = parts.expectedRedirectURI
				}

				assert.EqualValues(locHeader, expectedRedirect, "For webserver_type %s url %s expected redirect %s != actual %s", webserverType, reqURL, expectedRedirect, locHeader)
			}
		}
		err = app.Stop(true, false)
		assert.NoError(err)
	}

	// Change back to package dir. Lots of things will have to be cleaned up
	// in defers, and for windows we have to not be sitting in them.
	err = os.Chdir(packageDir)
	assert.NoError(err)
}

// TestMultipleComposeFiles checks to see if a set of docker-compose files gets
// properly loaded in the right order, with docker-compose.yaml first and
// with docker-compose.override.yaml last.
func TestMultipleComposeFiles(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	// Make sure that valid yaml files get properly loaded in the proper order
	app, err := ddevapp.NewApp("./testdata/testMultipleComposeFiles", true, "")
	assert.NoError(err)

	files, err := app.ComposeFiles()
	assert.NoError(err)
	require.NotEmpty(t, files)
	require.Equal(t, files[0], filepath.Join(app.AppConfDir(), "docker-compose.yaml"))
	require.Equal(t, files[len(files)-1], filepath.Join(app.AppConfDir(), "docker-compose.override.yaml"))

	// Make sure that some docker-compose.yml and docker-compose.yaml conflict gets noted properly
	app, err = ddevapp.NewApp("./testdata/testConflictingYamlYml", true, "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "there are more than one docker-compose.y*l")
	}

	// Make sure that some docker-compose.override.yml and docker-compose.override.yaml conflict gets noted properly
	app, err = ddevapp.NewApp("./testdata/testConflictingOverrideYaml", true, "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "there are more than one docker-compose.override.y*l")
	}

	// Make sure the error gets pointed out of there's no main docker-compose.yaml
	app, err = ddevapp.NewApp("./testdata/testNoDockerCompose", true, "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to find a docker-compose.yml or docker-compose.yaml")
	}

	// Catch if we have no docker files at all.
	// This should also fail if the docker-compose.yaml.bak gets loaded.
	app, err = ddevapp.NewApp("./testdata/testNoDockerFilesAtAll", true, "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to load any docker-compose.*y*l files")
	}
}

// TestGetAllURLs ensures the GetAllURLs function returns the expected number of URLs,
// and include the direct web container URLs.
func TestGetAllURLs(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s GetAllURLs", site.Name))

	testcommon.ClearDockerEnv()
	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)

	// Add some additional hostnames
	app.AdditionalHostnames = []string{"sub1", "sub2", "sub3"}

	err = app.WriteConfig()
	assert.NoError(err)

	err = app.StartAndWaitForSync(0)
	require.NoError(t, err)

	_, _, urls := app.GetAllURLs()

	// Convert URLs to map[string]bool
	urlMap := make(map[string]bool)
	for _, u := range urls {
		urlMap[u] = true
	}

	// We expect two URLs for each hostname (http/https) and two direct web container address.
	expectedNumUrls := len(app.GetHostnames())*2 + 2
	assert.Equal(len(urlMap), expectedNumUrls, "Unexpected number of URLs returned: %d", len(urlMap))

	// Ensure urlMap contains direct address of the web container
	webContainer, err := app.FindContainerByType("web")
	assert.NoError(err)
	require.NotEmpty(t, webContainer)

	expectedDirectAddress := app.GetWebContainerDirectHTTPSURL()
	if ddevapp.GetCAROOT() == "" {
		expectedDirectAddress = app.GetWebContainerDirectHTTPURL()
	}

	exists := urlMap[expectedDirectAddress]

	assert.True(exists, "URL list for app: %s does not contain direct web container address: %s", app.Name, expectedDirectAddress)

	// Multiple projects can't run at the same time with the fqdns, so we need to clean
	// up these for tests that run later.
	app.AdditionalFQDNs = []string{}
	app.AdditionalHostnames = []string{}
	err = app.WriteConfig()
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestWebserverType checks that webserver_type:apache-cgi or apache-fpm does the right thing
func TestWebserverType(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestWebserverType", site.Name))

		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		// Copy our phpinfo into the docroot of testsite.
		pwd, err := os.Getwd()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "servertype.php"), filepath.Join(app.AppRoot, app.Docroot, "servertype.php"))

		assert.NoError(err)
		for _, app.WebserverType = range []string{ddevapp.WebserverApacheFPM, ddevapp.WebserverApacheCGI, ddevapp.WebserverNginxFPM} {

			err = app.WriteConfig()
			assert.NoError(err)

			testcommon.ClearDockerEnv()

			startErr := app.StartAndWaitForSync(30)
			//nolint: errcheck
			defer app.Stop(true, false)
			if startErr != nil {
				appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
				assert.NoError(getLogsErr)
				t.Fatalf("app.StartAndWaitForSync failure; err=%v, logs:\n=====\n%s\n=====\n", startErr, appLogs)
			}
			out, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectHTTPURL()+"/servertype.php")
			require.NoError(t, err)

			expectedServerType := "Apache/2"
			if app.WebserverType == ddevapp.WebserverNginxFPM {
				expectedServerType = "nginx"
			}
			require.NotEmpty(t, resp.Header["Server"])
			require.NotEmpty(t, resp.Header["Server"][0])
			assert.Contains(resp.Header["Server"][0], expectedServerType, "Server header for project=%s, app.WebserverType=%s should be %s", app.Name, app.WebserverType, expectedServerType)
			assert.Contains(out, expectedServerType, "For app.WebserverType=%s phpinfo expected servertype.php to show %s", app.WebserverType, expectedServerType)
			err = app.Stop(true, false)
			assert.NoError(err)
		}

		// Set the apptype back to whatever the default was so we don't break any following tests.
		testVar := os.Getenv("DDEV_TEST_WEBSERVER_TYPE")
		if testVar != "" {
			app.WebserverType = testVar
			err = app.WriteConfig()
			assert.NoError(err)
		}
		runTime()
	}
}

// TestInternalAndExternalAccessToURL checks we can access content
// from host and from inside container by URL (with port)
func TestInternalAndExternalAccessToURL(t *testing.T) {
	assert := asrt.New(t)

	runTime := testcommon.TimeTrack(time.Now(), t.Name())

	site := TestSites[0]
	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)

	// Add some additional hostnames
	app.AdditionalHostnames = []string{"sub1", "sub2", "sub3"}
	app.AdditionalFQDNs = []string{"junker99.example.com"}

	for _, pair := range []testcommon.PortPair{{HTTPPort: "80", HTTPSPort: "443"}, {HTTPPort: "8080", HTTPSPort: "8443"}} {
		testcommon.ClearDockerEnv()
		app.RouterHTTPPort = pair.HTTPPort
		app.RouterHTTPSPort = pair.HTTPSPort
		err = app.WriteConfig()
		assert.NoError(err)

		// Make sure that project is absolutely not running
		err = app.Stop(true, false)
		assert.NoError(err)

		err = app.StartAndWaitForSync(5)
		assert.NoError(err)

		_, _, urls := app.GetAllURLs()

		// Convert URLs to map[string]bool
		urlMap := make(map[string]bool)
		for _, u := range urls {
			urlMap[u] = true
		}

		// We expect two URLs for each hostname (http/https) and two direct web container addresses.
		expectedNumUrls := len(app.GetHostnames())*2 + 2
		assert.Equal(len(urlMap), expectedNumUrls, "Unexpected number of URLs returned: %d", len(urlMap))

		_, _, URLList := app.GetAllURLs()
		URLList = append(URLList, "http://localhost", "http://localhost")
		for _, item := range URLList {
			// Make sure internal (web container) access is successful
			parts, err := url.Parse(item)
			assert.NoError(err)
			// Only try it if not an IP address URL; those won't be right
			hostParts := strings.Split(parts.Host, ".")

			// Make sure access from host is successful
			// But "localhost" is only for inside container.
			if parts.Host != "localhost" {
				_, _ = testcommon.EnsureLocalHTTPContent(t, item+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
			}

			if _, err := strconv.ParseInt(hostParts[0], 10, 64); err != nil {
				out, _, err := app.Exec(&ddevapp.ExecOpts{
					Service: "web",
					Cmd:     "curl -sS --fail " + item + site.Safe200URIWithExpectation.URI,
				})
				assert.NoError(err, "failed curl to %s: %v", item+site.Safe200URIWithExpectation.URI, err)
				assert.Contains(out, site.Safe200URIWithExpectation.Expect)
			}
		}
	}

	out, err := exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err)
	t.Logf("\n=========== output of ddev list ==========\n%s\n============\n", out)
	out, err = exec.RunCommand("docker", []string{"logs", "ddev-router"})
	assert.NoError(err)
	t.Logf("\n=========== output of docker logs ddev-router ==========\n%s\n============\n", out)

	// Set the ports back to the default was so we don't break any following tests.
	app.RouterHTTPSPort = "443"
	app.RouterHTTPPort = "80"
	app.AdditionalFQDNs = []string{}
	app.AdditionalHostnames = []string{}

	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestCaptureLogs checks that app.CaptureLogs() works
func TestCaptureLogs(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s CaptureLogs", site.Name))

	app := ddevapp.DdevApp{}

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	logs, err := app.CaptureLogs("web", false, "100")
	assert.NoError(err)

	assert.Contains(logs, "INFO spawned")

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestNFSMount tests ddev start functionality with nfs_mount_enabled: true
// This requires that the test machine must have NFS shares working
func TestNFSMount(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestNFSMount", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	app.NFSMountEnabled = true

	err = app.Start()
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, err)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "findmnt -T .",
	})
	assert.NoError(err)
	assert.Contains(stdout, ":"+dockerutil.MassageWindowsNFSMount(app.AppRoot))

	// Create a host-side dir symlink; give a second for it to sync, make sure it can be used in container.
	err = os.Symlink(".ddev", "nfslinked_.ddev")
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ls nfslinked_.ddev/config.yaml",
	})
	assert.NoError(err)

	// Create a host-side file symlink; give a second for it to sync, make sure it can be used in container.
	err = os.Symlink(".ddev/config.yaml", "nfslinked_config.yaml")
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ls nfslinked_config.yaml",
	})
	assert.NoError(err)

	// Create a container-side dir symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ln -s  .ddev nfscontainerlinked_ddev",
	})
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	assert.FileExists("nfscontainerlinked_ddev/config.yaml")

	// Create a container-side file symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ln -s  .ddev/config.yaml nfscontainerlinked_config.yaml",
	})
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	assert.FileExists("nfscontainerlinked_config.yaml")

	runTime()
	switchDir()
}

// TestWebcache tests ddev start functionality with webcache_enabled: true
func TestWebcache(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skipf("Skipping TestWebcache, not supported on %s", runtime.GOOS)
	}

	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestWebcache", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)
	app.WebcacheEnabled = true

	startErr := app.StartAndWaitForSync(1)
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, startErr)

	// Create a host-side dir symlink; give a second for it to sync, make sure it can be used in container.
	err = os.Symlink(".ddev", "webcachelinked_.ddev")
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ls webcachelinked_.ddev/config.yaml",
	})
	assert.NoError(err)

	// Create a container-side dir symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ln -s  .ddev webcachecontainerlinked_ddev",
	})
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	assert.FileExists("webcachecontainerlinked_ddev/config.yaml")

	// Create a container-side file symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ln -s  .ddev/config.yaml webcachecontainerlinked_config.yaml",
	})
	assert.NoError(err)
	time.Sleep(2 * time.Second)
	assert.FileExists("webcachecontainerlinked_config.yaml")

	runTime()
	switchDir()
}

// TestPortSpecifications tests to make sure that one project can't step on the
// ports used by another
func TestPortSpecifications(t *testing.T) {
	assert := asrt.New(t)
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprint("TestPortSpecifications"))
	defer runTime()
	testDir, _ := os.Getwd()

	site0 := TestSites[0]
	switchDir := site0.Chdir()
	defer switchDir()

	nospecApp := ddevapp.DdevApp{}
	err := nospecApp.Init(site0.Dir)
	assert.NoError(err)
	err = nospecApp.WriteConfig()
	require.NoError(t, err)
	// Since host ports were not explicitly set in nospecApp, they shouldn't be in globalconfig.
	require.Empty(t, globalconfig.DdevGlobalConfig.ProjectList[nospecApp.Name].UsedHostPorts)

	err = nospecApp.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer nospecApp.Stop(true, false)

	// Now that we have a working nospecApp with unspecified ephemeral ports, test that we
	// can't use those ports while nospecApp is running

	_ = os.Chdir(testDir)
	ddevDir, _ := filepath.Abs("./testdata/TestPortSpecifications/.ddev")

	specAppPath := testcommon.CreateTmpDir("specapp")
	//nolint: errcheck
	defer os.RemoveAll(specAppPath)
	err = fileutil.CopyDir(ddevDir, filepath.Join(specAppPath, ".ddev"))
	assert.NoError(err)

	specAPP, err := ddevapp.NewApp(specAppPath, false, "")
	assert.NoError(err)

	// It should be able to WriteConfig and Start with the configured host ports it came up with
	err = specAPP.WriteConfig()
	assert.NoError(err)
	err = specAPP.Start()
	assert.NoError(err)
	//nolint: errcheck
	err = specAPP.Stop(false, false)
	require.NoError(t, err)
	// Verify that DdevGlobalConfig got updated properly
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name])
	assert.NotEmpty(globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name].UsedHostPorts)

	// However, if we change change the name to make it appear to be a
	// different project, we should not be able to config or start
	conflictApp, err := ddevapp.NewApp(specAppPath, false, "")
	assert.NoError(err)
	conflictApp.Name = "conflictapp"

	err = conflictApp.WriteConfig()
	assert.Error(err)
	err = conflictApp.Start()
	assert.Error(err)

	// Now delete the specAPP and we should be able to use the conflictApp
	err = specAPP.Stop(true, false)
	assert.NoError(err)
	assert.Empty(globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name])

	err = conflictApp.WriteConfig()
	assert.NoError(err)
	err = conflictApp.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer conflictApp.Stop(true, false)
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[conflictApp.Name])
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[conflictApp.Name].UsedHostPorts)
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app *ddevapp.DdevApp) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	if container == nil {
		return "", fmt.Errorf("No container exists for containerType=%s app=%v", containerType, app)
	}
	name := dockerutil.ContainerName(*container)
	return name, nil
}

func removeAllErrCheck(path string, assert *asrt.Assertions) {
	err := os.RemoveAll(path)
	assert.NoError(err)
}
