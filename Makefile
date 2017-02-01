VERSION=0.1.0
NAME=fconf_$(VERSION)

build:
	gox  \
		-output "bin/{{.Dir}}/{{.OS}}_{{.Arch}}/{{.Dir}}_$(VERSION)/{{.Dir}}" \
		-osarch "linux/arm" github.com/FarmRadioHangar/fastc

tar:
	cd bin/ && tar -zcvf $(NAME).tar.gz  fastc/
