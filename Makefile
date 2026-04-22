demo:
	go run ./cmd/demo

compare:
	go run ./cmd/compare

writeup:
	pandoc writeup.md -o writeup.pdf --pdf-engine=xelatex -V geometry:margin=1in -V fontsize=11pt
