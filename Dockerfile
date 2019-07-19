FROM petronetto/php7-alpine
MAINTAINER "Zikani Nyirenda Mwase <zikani@nndi-tech.com>"
# Override directive from petronetto/php7-alpine and use the root user in the container
USER root 
RUN mkdir uploads
ADD composer.sh composer
RUN chmod +x composer
COPY composer-installer.sh composer-installer.sh
RUN chmod +x composer-installer.sh && ./composer-installer.sh
ENV COMPOSER_CACHE_DIR /tmp
# Thanks to: https://stackoverflow.com/a/35613430
# Symlink musl to the location where glibc should be since this is compiled on glibc, for now
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ADD ./dist/compozipd compozipd
EXPOSE 8025
CMD ["./compozipd",  "-u", "./uploads", "-h", ":8025" ]