package reporter

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

type jUnitTestSuites struct {
	XMLName    xml.Name        `xml:"testsuites"`
	TestSuites []jUnitTestSuite `xml:"testsuite"`
}

type jUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []jUnitTestCase `xml:"testcase"`
}

type jUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *jUnitFailure `xml:"failure,omitempty"`
}

type jUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// WriteJUnitXML outputs the fuzz run evaluations as standard JUnit XML for CI/CD test dashboards.
func WriteJUnitXML(filePath string, session *runner.SessionState) error {
	summary := session.Summary
	if summary == nil {
		return fmt.Errorf("session summary is nil")
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	suite := jUnitTestSuite{
		Name:      "ragfuzz_adversarial_suite",
		Tests:     session.Completed,
		Failures:  session.Failed,
		Errors:    0,
		Time:      fmt.Sprintf("%.3f", float64(session.DurationMs)/1000.0),
		Timestamp: session.StartTime.Format("2006-01-02T15:04:05Z07:00"),
	}

	for _, ev := range session.Evaluations {
		tc := jUnitTestCase{
			Classname: string(ev.TestCase.Vector),
			Name:      fmt.Sprintf("%s: %s", ev.TestCase.ID, ev.TestCase.MutatedQuery),
			Time:      fmt.Sprintf("%.3f", ev.Latency.Seconds()),
		}

		if !ev.Passed {
			tc.Failure = &jUnitFailure{
				Message: ev.FailureReason,
				Type:    "AdversarialVulnerabilityError",
				Content: fmt.Sprintf("Query: %s\nClean Query: %s\nTarget Chunk: %s\nRecall@K: %.2f\nGroundedness: %.2f\nDistractor Intrusion: %v\nAnswer: %s",
					ev.TestCase.MutatedQuery,
					ev.TestCase.CleanQuery,
					ev.TestCase.TargetChunkID,
					ev.MutatedRecallAtK,
					ev.Groundedness,
					ev.DistractorIntruded,
					ev.Response.Answer,
				),
			}
		}

		suite.TestCases = append(suite.TestCases, tc)
	}

	root := jUnitTestSuites{
		TestSuites: []jUnitTestSuite{suite},
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	fileContent := append([]byte(xml.Header), data...)
	return os.WriteFile(filePath, fileContent, 0644)
}
