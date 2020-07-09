rm -f aggregate-$stepID
wget https://storage.googleapis.com/tamabus-binary/aggregate-linux -O aggregate-$stepID
chmod +x aggregate-$stepID
./aggregate-$stepID
