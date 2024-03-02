
gen:
	go install . && \
    buf generate 

protoc-gen:
    go install . && \
	protoc -I=example \
    --go-setters=example \
    --go_out=example  \
	--go_opt=paths=source_relative \