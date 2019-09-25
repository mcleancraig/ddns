#!/bin/bash

# Set yo shit up!!
APIKEY="7a4gElr2E9mIwnMwLbSj"
NSONE_ZONE="fukka.co.uk"    # Zone according to nsone
AWS_ZONE="Z159W85ZCGY57D"   # Zone according to AWS
RECORD="fukka.co.uk"        # Record you want to change within the zone
IP_FINDER_LIST="http://bot.whatismyipaddress.com http://ipinfo.io/ip"
PATH=""                     # End with trailing / because I'm lazy
resolver="aws"
debug=false

# Get me the fuck out of here!
function bail() {
 echo "Bailing out:$1"
 exit 1
 }

# Make some NOISE!
function debug() {
[ $debug = "true" ] && echo "DEBUG: $1"
}

# Nicked from the interwebs, not by me but original author of this script - uncredited :(
function is_valid_ip()
{
 debug "running is_valid_ip $1"
  local ip=$1
  local stat=1 # set to fail to danger

  if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
    OIFS=$IFS
    IFS='.'
    ip=($ip)
    IFS=$OIFS
    [[ ${ip[0]} -le 255 && ${ip[1]} -le 255 && ${ip[2]} -le 255 && ${ip[3]} -le 255 ]]
      stat=$?
  fi
 debug "returining $stat from is_valid_ip"
  return $stat
}

#inspired by https://unix.stackexchange.com/questions/98923/programmatically-extract-private-ip-addresses
function is_public_ip()
{
 debug "running is_public_ip $1"
  local ip=$1
  local stat=2 # should never fail with this code
  
  echo $ip | grep -v -q -E '^(192\.168|10\.|172\.1[6789]\.|172\.2[0-9]\.|172\.3[01]\.)'
  stat=$?
  debug "returning $stat from is_public_ip"
  return $stat
}


# Loop through all the tools in IP_FINDER_LIST, Get my public IP, validate in and move on if we're happy or
# continue the loop if we're not.
get_current_ip() {
 debug "running get_current_ip for $RECORD"
 current_ip=$(dig +short $RECORD)
 debug "get_current_ip returning ${current_ip}"
}


get_public_ip(){
# Get my public IP
for finder in ${IP_FINDER_LIST}
do
  debug "checking from ${finder}"
  public_ip=$(wget -qO- ${finder})
  debug "returned ${public_ip}"
  if is_public_ip $public_ip && is_valid_ip $public_ip
  then
   break
  fi
done
}


# get our public IP and the record'dcurrent IP
get_public_ip || bail "Failed to run get_public_ip"
get_current_ip || bail "failed to run get_current_ip"

# Do we need to do anything?
[ "$current_ip" = "$public_ip" ] && bail "current ip for $RECORD is $current_ip, public ip is $public_ip, therefore no change required"

case $resolver in 
 nsone)
# NSOne version
 debug "calling NSONE to update IP record for $RECORD to $public_ip"
curl -X POST -H "X-NSONE-Key: $APIKEY" -d '{
 "answers": [
  {
   "answer": [
    "'$public_ip'"
   ]
  }
 ]
}' https://api.nsone.net/v1/zones/$NSONE_ZONE/$RECORD/A 
;;
 aws)
# AWS version 
 debug "calling AWS to update IP record for $RECORD to $public_ip"
 type ${PATH}aws >/dev/null 2>&1 || bail "aws cli not found in path. install it or add the path in the config!"
 ${PATH}aws route53 change-resource-record-sets --hosted-zone-id ${AWS_ZONE} --change-batch '{ "Comment": "ddns.sh called to update a record", "Changes": [ { "Action": "UPSERT", "ResourceRecordSet":{ "Name": "'$RECORD'", "Type": "A", "TTL": 100, "ResourceRecords": [ { "Value": "'$public_ip'" } ] } } ] }'  || bail "AWS Update Failed"
 ;;
  *) bail "Invalid resolver specified: ${resolver}" 
 ;;
esac

 
 
 
