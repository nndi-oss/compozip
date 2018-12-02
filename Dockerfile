FROM php:7.2-fpm-alpine
MAINTAINER "Zikani Nyirenda Mwase <zikani@nndi-tech.com>"
RUN mkdir -p /composer/{uploads,cache}
ENV COMPOSER_CACHE_DIR /composer/cache
EXPOSE 80
COPY composer-installer.sh /composer-installer.sh
RUN chmod +x /composer-installer.sh && /composer-installer.sh
ADD compozipd /
RUN chmod +x /compozipd
# Thanks to: https://stackoverflow.com/a/35613430
# Symlink musl to the location where glibc should be since this is compiled on glibc, for now
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
CMD ["/compozipd",  "-u", "/composer/uploads", "-h", ":80" ]