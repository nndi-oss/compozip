FROM php:7.2-fpm-alpine
MAINTAINER "Zikani Nyirenda Mwase <zikani@nndi-tech.com>"
RUN mkdir -p /composer/{uploads,cache}
ENV COMPOSER_CACHE_DIR /composer/cache
EXPOSE 80
COPY composer-installer.sh /composer-installer.sh
RUN chmod +x /composer-installer.sh && /composer-installer.sh
ADD compozipd /
ENTRYPOINT [ "compozipd", "-u", "/composer/uploads", "-h", ":8080"]