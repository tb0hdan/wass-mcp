package tools

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ToolsTestSuite struct {
	suite.Suite
}

func (s *ToolsTestSuite) TestBuildTargetURL_HTTP() {
	result := BuildTargetURL("localhost", 80)
	s.Equal("http://localhost:80", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_HTTPS() {
	result := BuildTargetURL("example.com", 443)
	s.Equal("https://example.com:443", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_CustomPort() {
	result := BuildTargetURL("example.com", 8080)
	s.Equal("http://example.com:8080", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_IPv6() {
	result := BuildTargetURL("::1", 443)
	s.Equal("https://[::1]:443", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_IPv6HTTP() {
	result := BuildTargetURL("::1", 80)
	s.Equal("http://[::1]:80", result)
}

func TestToolsTestSuite(t *testing.T) {
	suite.Run(t, new(ToolsTestSuite))
}
