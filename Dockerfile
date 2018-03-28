FROM python:3-alpine
ENV AWS_REGION eu-west-2
RUN apk add --no-cache ca-certificates 
ADD k8ecr /
ADD autodeploy.py /
RUN pip install requests 
ENV WEBHOOK https://hooks.slack.com/services/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
CMD ./autodeploy.py
