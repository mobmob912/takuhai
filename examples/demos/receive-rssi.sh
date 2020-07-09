rm -f receive-rssi-$stepID
wget https://storage.googleapis.com/tamabus-binary/receive-rssi -O receive-rssi-$stepID
chmod +x receive-rssi-$stepID
./receive-rssi-$stepID
