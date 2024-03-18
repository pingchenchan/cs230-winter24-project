From python:3.8-alpine
RUN pip3.8 install Flask kafka-python
WORKDIR /app
COPY . /app
CMD ["python", "app_v2.py"]
