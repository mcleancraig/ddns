package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	api "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func main() {
	err := getConfig()
	if err != nil {
		log.Fatal(err)
	}
	var (
		debug = viper.GetString("debug")
	)

	if debug == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debug("In main")
	err = compareAndRun()
	if err != nil {
		log.Fatal(err)
	} else {
		logrus.Info("Completed")
	}
}
func getConfig() (err error) {
	logrus.Debug("In function getConfig")
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.AddConfigPath("/etc/ddns/")
	err = viper.ReadInConfig()
	if err != nil {
		return err // not strictly needed as this is all it will return by default
	}
	return nil
}

func compareAndRun() (err error) {
	logrus.Debug("In function compareAndRun")
	currentIP, err := getCurrentIP()
	if err != nil {
		return errors.Errorf("Failed getting IP: %v", err)
	}
	currentDNS, err := getCurrentDNS()
	if err != nil {
		return errors.Errorf("DNS Lookup failed: %v", err)
	}
	if currentIP == nil {
		return errors.New("Current IP is not valid")
	}
	if currentDNS == nil {
		return errors.New("Current DNS is not valid")
	}
	if string(currentDNS) == string(currentIP) {
		logrus.Info("IPs match - no action required")
		return nil
	}
	logrus.Info("ip doesn't match dns - update wanted")
	err = changeIP(currentIP)
	if err != nil {
		return err
	}
	return nil
}

func changeIP(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeIP")
	switch viper.GetString("provider") {
	case "aws":
		err = changeAWS(requestedIP)
		if err != nil {
			logrus.Error("provider ChangeAWS returned error: %s", err)
			return err
		}
	case "nsone":
		err = changeNSONE(requestedIP)
		if err != nil {
			logrus.Error("provider ChangeNSONE returned error: %s", err)
			return err
		}
	default:
		logrus.Error("provider not set in config, or set to incorrect value")
		return errors.New("provider not set")
	}
	return nil
}

func getCurrentIP() (reportedIP net.IP, err error) {
	logrus.Debug("In function getCurrentIP")
	for _, v := range viper.GetStringSlice("ip_finder") {
		logrus.Debugf("Getting IP from %s", v)
		resp, err := http.Get(v)
		if err != nil {
			logrus.Errorf("from %s: %v", v, err)
		} else {
			defer resp.Body.Close()
			logrus.Debugf("Reading response from %s", v)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("from %s: %v", v, err)
				return nil, err
			}
			reportedIP := net.ParseIP(strings.TrimSpace(string(body)))
			logrus.Debugf("response from %v was: %v ", v, reportedIP)
			if reportedIP != nil {
				logrus.Infof("Current IP address reported by %v as: %v ", v, reportedIP)
				return reportedIP, nil

			} else {
				logrus.Errorf("Missed getting an IP from %s", v)
				return nil, err
			}

		}
	}
	return net.ParseIP(""), errors.New("get IP failed")
}

func getCurrentDNS() (_ net.IP, err error) {

	logrus.Debug("In function getCurrentDns")
	logrus.Debug("Finding nameservers for ", viper.GetString("record"))

	//
	// ns method
	//

	nameserver, _ := net.LookupNS(viper.GetString("record"))
	if nameserver == nil {
		return nil, errors.Errorf("No nameservers found for %v", viper.GetString("record"))
	}
	for _, ns := range nameserver {
		nshost := ns.Host
		logrus.Debugf("Looking up %v against %v", viper.GetString("record"), nshost)
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp4", net.JoinHostPort(nshost, "53"))
			},
		}
		resp, err := resolver.LookupIP(context.Background(), "ip4", viper.GetString("record"))
		if err != nil {
			return nil, errors.Errorf("error from resolver: %v", err)
		}
		// Now we get an array back, so we need to find out what's in it
		for _, name := range resp {
			outputIP := net.ParseIP(name.String())
			logrus.Infof("DNS Lookup returned %s from %v", outputIP, nshost)
			return outputIP, nil
		}

	}

	return nil, errors.New("Fell out of nameserver loop without getting any results")

}

func changeAWS(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeAWS")

	sess, err := session.NewSession()
	if err != nil {
		return errors.Errorf("Failed to open AWS session: %v", err)
	}
	svc := route53.New(sess)
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: aws.String(viper.GetString("record")), // Required
						Type: aws.String("A"),                       // Required
						TTL:  aws.Int64(600),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(requestedIP.String()), // Required
							},
						},
					},
				},
			},
			Comment: aws.String("Changed by ddns script"),
		},
		HostedZoneId: aws.String(viper.GetString("awsZone")), // Required
	}

	_, err = svc.ChangeResourceRecordSets(params)
	if err != nil {
		return errors.Errorf("AWS Failed to update!:\n %v", err)
	}
	return nil
}

func changeNSONE(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeNSONE")

	httpClient := &http.Client{Timeout: time.Second * 10}
	client := api.NewClient(httpClient, api.SetAPIKey(viper.GetString("api_key")))
	zone := viper.GetString("nsone_zone")
	domain := viper.GetString("record")
	logrus.Debug("Looking for " + domain + " in zone " + zone)

	record, _, err := client.Records.Get(zone, domain, "A")
	if err != nil {
		log.Fatal(err)
	}

	logrus.Debugf("found %T: %v", record, record)
	logrus.Debugf("changing to %v", requestedIP)
	record.Answers = []*dns.Answer{
		dns.NewAv4Answer(requestedIP.String())}
	_, err = client.Records.Update(record)
	logrus.Debugf("pushing change")

	if err != nil {
		return errors.Errorf("NSOne failed to update!:\n %v", err)
	}
	return nil
}
