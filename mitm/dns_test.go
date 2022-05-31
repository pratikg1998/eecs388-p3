package main

import (
	"net"
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

func dnsWithDomainQuestions(domains []string) *layers.DNS {
	questions := make([]layers.DNSQuestion, 0, len(domains))
	for _, d := range domains {
		questions = append(questions, layers.DNSQuestion{
			Name:  []byte(d),
			Type:  layers.DNSTypeA,
			Class: layers.DNSClassIN,
		})
	}

	return &layers.DNS{
		QR:        false,
		OpCode:    layers.DNSOpCodeQuery,
		QDCount:   uint16(len(questions)),
		Questions: questions,
	}
}

func TestHasQuestionForDomain(t *testing.T) {
	for _, v := range []struct {
		name      string
		questions []string
		domain    string
		expected  bool
	}{
		// We're stuttering on "packet with", but it does make the output clearer
		// for a reader without context.
		{"packet with no questions", nil, "eecs388.org", false},
		{"packet with correct domain", []string{"eecs388.org"}, "eecs388.org", true},
		{"packet with other correct domain", []string{"test.domain"}, "test.domain", true},
		{"packet with different domain", []string{"wrong.com"}, "eecs388.org", false},
		{"packet with prefix of correct domain", []string{"eecs388.orgcom"}, "eecs388.org", false},
	} {
		v := v
		t.Run(v.name, func(t *testing.T) {
			got := HasQuestionForDomain(dnsWithDomainQuestions(v.questions), v.domain)
			assert.Equal(t, v.expected, got, "HasQuestionForDomain(dns, %q) returned incorrect value", v.domain)
		})
	}
}

func TestAnswerForQuestion(t *testing.T) {
	domain := []byte("eecs388.org")
	ip := net.ParseIP("3.23.25.235")

	answer := AnswerForQuestion(layers.DNSQuestion{
		Name:  domain,
		Type:  layers.DNSTypeA,
		Class: layers.DNSClassIN,
	}, ip)

	assert.EqualValues(t, domain, answer.Name, "got wrong name in answer. The name tells the client which domain this answer is for!", domain, answer.Name)
	assert.Equal(t, layers.DNSTypeA, answer.Type, "got unexpected resource record type. Remember that we only deal with A-type queries in this project! Check the type of layers.DNS.Type in the package's documentation for a further pointer.")
	assert.Equal(t, layers.DNSClassIN, answer.Class, "got unexpected resource record class. We only deal with the internet in this project! Check the type of layers.DNS.Class in the package's documentation for a further pointer.")
	assert.True(t, answer.IP.Equal(ip), "expected IP %s in answer, got %s", ip, answer.IP)
}
