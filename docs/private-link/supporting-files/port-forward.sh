#!/bin/bash

# Sync clock otherwise AWS CLI will fail
sudo hwclock --hctosys

# check arguments
if [ $# != 1 ]; then
   echo "[Error] Specify 1 or 2 as an argument to claim which Terminal"
   echo "[Error] How to use:  double-port-forward.sh 1|2"
   exit 1
fi
if [ $1 = "1" ]; then
   TERMINAL="1"
   echo "[Log] Set up as Terminal 1. This will log in public bation." 
elif [ $1 = "2" ]; then
   TERMINAL="2"
   echo "[Log] set up as Terminal 2 (assuming you alread have another seesion by Terminal 1) This will login in private bastion" 
else
   echo "[Error] Specify 1 or 2 as an argument to claim which Terminal"
   echo "[Error] How to use:  double-port-forward.sh  1|2"
   exit 1
fi

# -----------------------------------------------------------------------
# Get private key from AWS
# 1. Configure AWS CLI
echo "[Log] 1. aws conigure"
# aws configure

# 2. Get key ID from Systems Manager paramater
echo "[Log] 2. Search KyePair Id from System Manager Parameter"
KeyPairID=`aws ec2 describe-key-pairs --key-names "BastionKeyPair" --query 'KeyPairs[].KeyPairId' --output text`

# 3. Download key from Systems Manager parameter
echo "[Log] 3. download ssh key from System Manager Parameter using keypair id"
echo "[Log] 3-1. delete old bastion-key.pem just in case"
rm -f  bastion-key.pem
echo "[Log] 3-2. download new bastion-key.pem from AWS"
echo "$(aws ssm get-parameter --name /ec2/keypair/${KeyPairID} --with-decryption --query Parameter.Value --output text)" > bastion-key.pem

# RESULT=`aws ssm get-parameter --name /ec2/keypair/${KeyPairID} --with-decryption --query Parameter.Value --output text`
#echo $?
# if [ $? != 0 ]; then
#   echo $RESULT > bastion-key.pem
# else
#   echo "[Error] Something is wrong"
#   exit 255
# fi

# 4. Set appropriate file mode bits
chmod 400 bastion-key.pem
echo "[Log] 4. bastion-key.pem to 400"
echo "Bastion key is downloaded as bastion-key.pem" 

# Get Bastion IPS
echo "[Log] 5. Get Bastion IPs from AWS"
export PUBLIC_BASTION=`aws ec2 describe-instances | jq -r '.Reservations[] | [.Instances[].InstanceType, .Instances[].PrivateIpAddress, .Instances[].PublicIpAddress, .Instances[].Tags[]?.Value ]  | @csv' | egrep "Public" | awk -F'[,]' '{print $3}'  |  grep -v -e '^\s*$' | sed 's/"//g'`
echo "[Log] 5-1. Public Bastion IP is $PUBLIC_BASTION"
export PRIVATE_BASTION=`aws ec2 describe-instances | jq -r '.Reservations[] | [.Instances[].InstanceType, .Instances[].PrivateIpAddress, .Instances[].PublicIpAddress, .Instances[].Tags[]?.Value ]  | @csv' | grep "Private" | awk -F'[,]' '{print $2}' | grep -v -e '^\s*$' | sed 's/"//g'`
echo "[Log] 5-2. Private Bastion IP is $PRIVATE_BASTION"


# 5. Copy to Bastion server in Public zone ( assume there is only one instance that has public IP)
echo "[Log] 6. copy ssh key to bation. Access to bastion."

# ---------------------------------------------------------------------------------- 
if [ $TERMINAL = "1" ]; then
  # Terminal 1
  echo "[Log] 6-1. Login as Terminal 1" 
  ssh -i bastion-key.pem -p 22 ec2-user@$PUBLIC_BASTION -L 10022:$PRIVATE_BASTION:22
  echo "[Log] NOTICE: You need to add your VPC to Route53 zone as a related VPC so that you can resolve domain names created by ROSA from this bastion."
else 
  # Terminal 2
  echo "[Log] 6-1. Login as Terminal 2 (assuming you already have Terminal 1 session)"
  ssh-keygen -f ~/.ssh/known_hosts -R "[localhost]:10022"
  ssh -i bastion-key.pem -p 10022 ec2-user@localhost -D 10044
  echo "[Log] NOTICE: You need to add your VPC to Route53 zone as a related VPC so that you can resolve domain names created by ROSA from this bastion."
fi

# Local /etc/hosts
# console-openshift-console..... = console
# oauth-openshift.... = OAuth server
# The IP is private IP. Find the IP in the private network using "dig 
     +short"
# 10.0.1.214    console-openshift-console.apps.singleaz.dzfa.p1.openshiftapps.com
# 10.0.1.214    oauth-openshift.apps.singleaz.dzfa.p1.openshiftapps.com

   

