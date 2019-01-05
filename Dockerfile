FROM scratch
ADD build/app_cgo /app
CMD ["/app"]
