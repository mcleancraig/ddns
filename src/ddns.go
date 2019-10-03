//TODO
// Do authoritative DNS lookup rather than just address?

package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

func main() {
	var err error
	err = getConfig()
	if err != nil {
		log.Fatal(err)
	}
	if viper.GetString("debug") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debug("In main")
	err = compareAndRun()
	if err != nil {
		log.Fatal(err)
	}
}

func compareAndRun() (err error) {
	logrus.Debug("In function compareAndRun")
	currentIP, err := getCurrentIp()
	if err != nil {
		return
	}
	currentDNS, err := getCurrentDns()
	if err != nil {
		return
	}
	if currentIP == nil {
		return errors.New("Current IP is not valid")
	}
	if currentDNS == nil {
		return errors.New("Current DNS is not valid")
	}
	for dns_resp := range currentIP {
		//if string(currentIP) == string(currentDNS) {
		if string(currentIP) == string(dns_resp) {
			logrus.Info("IPs match - no action required")
			return nil
		}
	}
	logrus.Info("ip doesn't match dns - update wanted")
	err = changeIP(currentIP)
	if err != nil {
		return
	} else {
		return nil
	}

}

func changeIP(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeIP")
	resolver := viper.GetString("resolver")
	switch resolver {
	case "aws":
		err = changeAWS(requestedIP)
		if err != nil {
			return
		}
	case "nsone":
		changeNSONE()
	default:
		return errors.New("resolver not set in config, or set to incorrect value")
	}
	return errors.New("we should never get here, fell off the switch in changeIP")
}

func getConfig() (err error) {
	logrus.Debug("In function getConfig")
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.AddConfigPath("/etc/ddns/")
	err = viper.ReadInConfig()
	if err != nil {
		return
		// logrus.Errorf("%v", err)
	}
	return nil
}

func getCurrentIp() (reportedIP net.IP, err error) {
	logrus.Debug("In function getCurrentIP")
	var (
		finder = viper.GetStringSlice("ip_finder")
	)
	for _, v := range finder {
		resp, err := http.Get(v)
		if err != nil {
			logrus.Errorf("from %s: %v", v, err)
		} else {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("from %s: %v", v, err)
			}
			reportedIp := net.ParseIP(strings.TrimSpace(string(body)))
			logrus.Debugf("response from %v was: %v ", v, string(body))
			if reportedIp != nil {
				logrus.Infof("Current IP address reported by %v as: %v ", v, reportedIp)

				return reportedIp, nil

			}

		}
	}
	return net.ParseIP(""), errors.New("get IP failed")
}

func getCurrentDns() (_ []net.IPAddr, err error) {
	// need to do authoritative lookup here, avoid cache

	// lookup NS servers
	// fall out if no NS records found
	// loop through servers
	// lookup target on server
	// return if found
	// fall out if not
	logrus.Debug("In function getCurrentDns")
	nameservers, err := net.LookupNS(viper.GetString("record"))
	if err != nil {
		return nil, err
	}
	for _, ns := range nameservers {
		nshost := ns.Host
		logrus.Debugf("Looking up %v against %v", viper.GetString("record"), nshost)
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", net.JoinHostPort(nshost, "53"))
			},
		}
		resp, err := resolver.LookupIPAddr(context.Background(), viper.GetString("record"))
		if err != nil {
			return nil, errors.Errorf("error from resolver: %v", err)
		}
		return resp, nil

	}
	return nil, errors.New("Fell out of nameserver loop without getting any results")
	//currentAddress, _ := net.LookupIP(viper.GetString("record"))
	//for _, ip := range currentAddress {
	//	logrus.Info("current DNS reported as " + ip.String())
	//	return net.ParseIP(ip.String()), nil
	//}
	//logrus.Fatal("fell out of the loop")
	//return net.ParseIP("Failed"), errors.New("DNS lookup failed")

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
				{ // Required
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: aws.String(viper.GetString("record")), // Required
						Type: aws.String("A"),                       // Required
						TTL:  aws.Int64(600),
						ResourceRecords: []*route53.ResourceRecord{
							{ // Required
								Value: aws.String(string(requestedIP)), // Required
							},
						},
					},
				},
			},
			Comment: aws.String("Changed by ddns script"),
		},
		HostedZoneId: aws.String(viper.GetString("aws_zone")), // Required
	}

	_, err = svc.ChangeResourceRecordSets(params)
	if err != nil {
		return errors.Errorf("AWS Failed to update!:\n %v", err)
	}
	return nil
}

func changeNSONE() {
	logrus.Debug("This needs to be written!")
}
