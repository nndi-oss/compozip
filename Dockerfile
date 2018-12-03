FROM php:7.2-fpm-alpine
MAINTAINER "Zikani Nyirenda Mwase <zikani@nndi-tech.com>"
ADD composer.sh /bin/composer
ADD compozipd /
RUN mkdir /uploads \
  && echo '{"name":"compozip/dummy","require":{"php":">=5.6.30"}}' > /uploads/composer.json \
  && chmod +x /bin/composer
COPY composer-installer.sh /composer-installer.sh
RUN chmod +x /composer-installer.sh && /composer-installer.sh
ENV COMPOSER_CACHE_DIR /tmp
# Thanks to: https://stackoverflow.com/a/35613430
# Symlink musl to the location where glibc should be since this is compiled on glibc, for now
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
EXPOSE 80
CMD ["/compozipd",  "-u", "/uploads", "-h", ":80" ]