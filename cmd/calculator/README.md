# calculator-go
An application that calculates the confidence score for a given piece of data

## Summary of Data Confidence scoring algorithm
1. Establish the total weighting of all factors
    - Defined by policy where AnnotationType = value from 1-5

      `Example: "tpm"=2`

    - If no policy or factor not found in policy, default weight will always be 1
    - For this working example, assume the following annotation types and weights
      ```
      tpm=2
      
      tls=1

      pki=1
      ```
    - Resulting total weights for the above where all annotations are present == 4

2. Establish which annotations are satisfied
    - Treat IsSatisfied == true as a 1 and multiply by the annotation's weight
    - For this example, TPM and TLS will be satisfied
      ```
      tpm -- IsSatisfied (1) Weight (2) == 2
      
      tls -- IsSatisfied (1) Weight (1) == 1
      
      pki -- IsSatisfied (0) Weight (1) == 0
      ```
    - Resulting total weight for the satisfied annotations == 3


3. Divide satisfied weight score by total weight score
    - 3 / 4 = .75 (%75 confidence)

## Steps to Run OPA as server in docker container

1. Execute the following command inside the root directory of the project to build docker image from `Dockerfile`
    ```bash
    docker build -t opa-server scripts/policies
    ```

2. Execute the following command to `opa-server` container
    ```bash
    docker run --publish 8181:8181 opa-server
    ```

## Steps to fetch the weights from OPA
1. Create `input.json` with the following content
    ```json
    {"input":{"class":"production"}}
    ```

2. To execute a curl request using `input.json`
    ```bash
    curl localhost:8181/v1/data/dcf_scoring -d @input.json -H 'Content-Type: application/json' | jq
    ```

3. The response should be similar to
    ```json
    {
      "result": {
        "weights": [
          {
            "pki": 2,
            "tls": 2,
            "tpm": 1
          }
        ]
      }
    }
    ```