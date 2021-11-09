FROM busybox
COPY ./out /out
CMD ["/out"]