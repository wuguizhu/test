#!/bin/bash
echo ".........start to deploy  ............"
echo "download master.zip"
rm -rf ./master.zip*
wget https://github.com/wuguizhu/test/archive/master.zip
unzip -o master.zip
cd test-master
./testnode-pinger &
echo " testnode-pinger is running successfully"