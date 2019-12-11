#!/bin/bash
echo ".........start to deploy  ............"
echo "download master.zip"
rm -rf ./master.zip*
wget https://github.com/wuguizhu/test/archive/master.zip
unzip -o master.zip
cd test-master
chmod 777 testnode-pinger
./testnode-pinger & 
if [ $? -ne 0 ]; then
    echo "testnode-pinger deploy failed!"
else
    echo " testnode-pinger is running successfully"
fi
