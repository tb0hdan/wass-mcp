package tools

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/tb0hdan/wass-mcp/pkg/types"
)

type ToolsTestSuite struct {
	suite.Suite
}

// ParseHostInput tests.

func (s *ToolsTestSuite) TestValidateInput_HostnameStartingWithDigit() {
	bs := NewBaseScanner("test", "test", zerolog.Nop())
	input := ScannerInput{Host: "0x21h.net"}
	input = bs.PrepareInput(input)
	s.NoError(bs.ValidateInput(&input))
}

func (s *ToolsTestSuite) TestParseHostInput_PlainHostname() {
	result := ParseHostInput("example.com")
	s.Equal("example.com", result.Host)
	s.Equal("", result.Scheme)
	s.Equal(0, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_PlainIP() {
	result := ParseHostInput("192.168.1.1")
	s.Equal("192.168.1.1", result.Host)
	s.Equal("", result.Scheme)
	s.Equal(0, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_Empty() {
	result := ParseHostInput("")
	s.Equal("", result.Host)
	s.Equal("", result.Scheme)
	s.Equal(0, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_HTTPSUrl() {
	result := ParseHostInput("https://example.com")
	s.Equal("example.com", result.Host)
	s.Equal("https", result.Scheme)
	s.Equal(0, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_HTTPUrl() {
	result := ParseHostInput("http://example.com")
	s.Equal("example.com", result.Host)
	s.Equal("http", result.Scheme)
	s.Equal(0, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_URLWithPort() {
	result := ParseHostInput("https://example.com:8443")
	s.Equal("example.com", result.Host)
	s.Equal("https", result.Scheme)
	s.Equal(8443, result.Port)
}

func (s *ToolsTestSuite) TestParseHostInput_URLWithPath() {
	result := ParseHostInput("https://example.com/path")
	s.Equal("example.com", result.Host)
	s.Equal("https", result.Scheme)
	s.Equal(0, result.Port)
}

// ResolveParams tests.

func (s *ToolsTestSuite) TestResolveParams_Defaults() {
	params := ResolveParams(ScannerInput{})
	s.Equal(types.DefaultHost, params.Host)
	s.Equal(types.DefaultPort, params.Port)
	s.Equal(types.SchemeHTTP, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_PlainHost() {
	params := ResolveParams(ScannerInput{Host: "example.com", Port: 8080})
	s.Equal("example.com", params.Host)
	s.Equal(8080, params.Port)
	s.Equal(types.SchemeHTTP, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_Port443InfersHTTPS() {
	params := ResolveParams(ScannerInput{Host: "example.com", Port: 443})
	s.Equal("example.com", params.Host)
	s.Equal(443, params.Port)
	s.Equal(types.SchemeHTTPS, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_HTTPSUrl() {
	params := ResolveParams(ScannerInput{Host: "https://example.com"})
	s.Equal("example.com", params.Host)
	s.Equal(443, params.Port)
	s.Equal(types.SchemeHTTPS, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_HTTPSUrlWithPort() {
	params := ResolveParams(ScannerInput{Host: "https://example.com:8443"})
	s.Equal("example.com", params.Host)
	s.Equal(8443, params.Port)
	s.Equal(types.SchemeHTTPS, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_HTTPUrl() {
	params := ResolveParams(ScannerInput{Host: "http://example.com"})
	s.Equal("example.com", params.Host)
	s.Equal(80, params.Port)
	s.Equal(types.SchemeHTTP, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_ExplicitPortOverridesURLPort() {
	params := ResolveParams(ScannerInput{Host: "https://example.com:8443", Port: 9999})
	s.Equal("example.com", params.Host)
	s.Equal(9999, params.Port)
	s.Equal(types.SchemeHTTPS, params.Scheme)
}

func (s *ToolsTestSuite) TestResolveParams_Vhost() {
	params := ResolveParams(ScannerInput{Host: "192.168.1.1", Port: 80, Vhost: "test.com"})
	s.Equal("test.com", params.Vhost)
}

// BuildTargetURL tests.

func (s *ToolsTestSuite) TestBuildTargetURL_HTTP() {
	result := BuildTargetURL(ScanParams{Host: "localhost", Port: 80, Scheme: types.SchemeHTTP})
	s.Equal("http://localhost", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_HTTPS() {
	result := BuildTargetURL(ScanParams{Host: "example.com", Port: 443, Scheme: types.SchemeHTTPS})
	s.Equal("https://example.com", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_CustomPort() {
	result := BuildTargetURL(ScanParams{Host: "example.com", Port: 8080, Scheme: types.SchemeHTTP})
	s.Equal("http://example.com:8080", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_HTTPSCustomPort() {
	result := BuildTargetURL(ScanParams{Host: "example.com", Port: 8443, Scheme: types.SchemeHTTPS})
	s.Equal("https://example.com:8443", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_IPv6() {
	result := BuildTargetURL(ScanParams{Host: "::1", Port: 443, Scheme: types.SchemeHTTPS})
	s.Equal("https://[::1]", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_IPv6HTTP() {
	result := BuildTargetURL(ScanParams{Host: "::1", Port: 80, Scheme: types.SchemeHTTP})
	s.Equal("http://[::1]", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_IPv6CustomPort() {
	result := BuildTargetURL(ScanParams{Host: "::1", Port: 8080, Scheme: types.SchemeHTTP})
	s.Equal("http://[::1]:8080", result)
}

func (s *ToolsTestSuite) TestBuildTargetURL_EmptySchemeDefaultsHTTP() {
	result := BuildTargetURL(ScanParams{Host: "example.com", Port: 80})
	s.Equal("http://example.com", result)
}

func TestToolsTestSuite(t *testing.T) {
	suite.Run(t, new(ToolsTestSuite))
}
