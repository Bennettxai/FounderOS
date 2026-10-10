package email

import (
	"strings"
	"testing"
)

// The addresses from the 2026-08-21 incident (an earlier incident): the exact sends this
// guard exists to have stopped.
const customer = "customer@client.example"

var invented = []string{"alex@agency.example", "sam@agency.example", "jo@creator.example"}

func noEnv(string) string { return "" }

func blocked(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected the send to be refused")
	}
	return err.Error()
}

func TestMailGuardBlocksTheCustomerAddress(t *testing.T) {
	_, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: customer}, noEnv)
	if msg := blocked(t, err); !strings.Contains(msg, customer) {
		t.Errorf("error should name the recipient: %s", msg)
	}
}

func TestMailGuardBlocksEveryInventedAddress(t *testing.T) {
	for _, a := range invented {
		_, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: a}, noEnv)
		blocked(t, err)
	}
}

func TestMailGuardBlocksExternalHiddenInCc(t *testing.T) {
	_, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: "alex@personal.example", Cc: customer}, noEnv)
	if msg := blocked(t, err); !strings.Contains(msg, "1 external recipient(s)") {
		t.Errorf("msg = %s", msg)
	}
}

func TestMailGuardBlocksSendingAsTheSystemAddress(t *testing.T) {
	_, err := CheckOutboundMail(OutboundMail{From: "OS@FounderOS.local", To: "alex@personal.example"}, noEnv)
	if msg := blocked(t, err); !strings.Contains(msg, "2026-08-18") {
		t.Errorf("msg = %s", msg)
	}
	env := func(k string) string {
		if k == "MAIL_ALLOW_SYSTEM_FROM" {
			return "1"
		}
		return ""
	}
	if _, err := CheckOutboundMail(OutboundMail{From: "os@founderos.local", To: "alex@personal.example"}, env); err != nil {
		t.Errorf("per-call override should allow it: %v", err)
	}
}

func TestMailGuardAllowsInternalMail(t *testing.T) {
	got, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: "Alex@LaunchpadCohort.example, alex@personal.example"}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "alex@launchpadcohort.example,alex@personal.example" {
		t.Errorf("recipients = %v", got)
	}
}

func TestMailGuardAllowsExternalOnlyWithOverride(t *testing.T) {
	env := func(k string) string {
		if k == "MAIL_ALLOW_EXTERNAL" {
			return "1"
		}
		return ""
	}
	if _, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: customer}, env); err != nil {
		t.Fatal(err)
	}
}

func TestMailGuardDisplayNameCannotSmuggleARecipient(t *testing.T) {
	_, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: "alex@vantage.example <" + customer + ">"}, noEnv)
	blocked(t, err)
}

func TestMailGuardRefusesAnEmptyRecipientList(t *testing.T) {
	_, err := CheckOutboundMail(OutboundMail{From: "alex@vantage.example", To: " , "}, noEnv)
	if msg := blocked(t, err); msg != "no recipient address" {
		t.Errorf("msg = %s", msg)
	}
}
