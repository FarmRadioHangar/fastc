VERSION=0.1.5
NAME=fastc_$(VERSION)

build:
	gox  \
		-output "bin/{{.Dir}}_$(VERSION)/{{.OS}}_{{.Arch}}/{{.Dir}}" \
		-osarch "linux/arm" github.com/FarmRadioHangar/fastc

copy-tpl:
	cp extensions_additional.conf.fastc bin/fastc_${VERSION}

tar: copy-tpl
	cd bin/ && tar -zcvf $(NAME).tar.gz  fastc_${VERSION}/
