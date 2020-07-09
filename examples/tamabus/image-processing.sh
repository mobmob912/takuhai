
echo 'shell tamabus receive'
rm -f image-processing
wget https://storage.googleapis.com/tamabus-binary/image-processing
echo 'wgeted'
chmod +x image-processing
echo 'chmod'
./image-processing
