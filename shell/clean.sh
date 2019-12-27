#!/bin/bash
echo ".........start to clean up............"
rm -rf ../test-master
echo "cleaning up /test/ folders finished..."
rm -rf ../master.zip*
rm -f ../deploy.sh*
cd ..
echo "cleaning up zip files finished........"
echo ".........finish clean up.............."
