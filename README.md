# go-multi-threaded-up-down-load-api
#### Installation and docker deployment instructions ########
1. Clone the repository  code,  run: ``git clone https://github.com/NderKev/go-multi-threaded-up-down-load-api.git``
2. Navigate to the main directory,  run: ``cd go-multi-threaded-up-down-load``
3. If the project isn't initialized with Go modules, run:  ``go init go-multi-threaded-up-down-load-app``
3. Clean up unnecessary dependencies and ensure all required ones are listed, run: ``go mod tidy``
4. Then add the pq PostgreSQL driver dependency using : ``go get github.com/lib/pq`` and ``go get github.com/jackc/pgx/v4``
5. Dockerize build and run the containers ``docker-compose up --build``
6. Test app database connection  ```curl http://localhost:8989``
7. Use the cURL commands or Postman links and command to interact with the APIs:
      . Upload: POST /upload with file, fileID, and partID form data. 
      . Get Metadata: GET /getdata?fileID=1.
      . Download: GET /download?fileID=1.