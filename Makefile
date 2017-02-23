VERSION=0.1.3
NAME=fastc_$(VERSION)

build:
	gox  \
		-output "bin/{{.Dir}}_$(VERSION)/{{.OS}}_{{.Arch}}/{{.Dir}}" \
		-osarch "linux/arm" github.com/FarmRadioHangar/fastc

tar:
	cd bin/ && tar -zcvf $(NAME).tar.gz  fastc_${VERSION}/
