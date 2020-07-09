rm -f store-image-$stepID
wget https://storage.googleapis.com/tamabus-binary/store-busdata -O store-image-$stepID
chmod +x store-image-$stepID
./store-image-$stepID
