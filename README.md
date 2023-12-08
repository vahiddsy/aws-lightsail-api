---

# AWS Lightsail Go Application

This Go application uses the AWS SDK v2 to interact with AWS Lightsail instances. It provides HTTP endpoints to list instances and reset a specific instance.

## Prerequisites

Before running this application, you need to have the following in place:

1. AWS Access Key and Secret Key: You must have AWS IAM credentials with access to Lightsail services. Setting the environment variable your shell session.

    ```sh
    export AWS_ACCESS_KEY_ID="-----"
    export AWS_SECRET_ACCESS_KEY="-------------"
    ```

2. Go Installation: You need Go installed on your machine. You can download and install it from the [official Go website](https://golang.org/dl/).

3. AWS CLI (Optional): To verify and manage your AWS credentials, you can install the AWS Command Line Interface (CLI) from the [AWS CLI documentation](https://aws.amazon.com/cli/).

4. Go Dependencies: The application relies on Go modules and AWS SDK v2. To install the required dependencies, navigate to the project directory and run:

   ```sh
   go mod download
   ```

## Usage

### Running the Application

To run the application, execute the following command in your terminal from the project directory:

```sh
go run api-handler.go
```

The application will start an HTTP server on port 8080.

### Listing Lightsail Regions

To list Lightsail Regions, make a GET request to the following endpoint:

```
http://localhost:8080/api/regions
```


### Listing Lightsail Instances

To list Lightsail instances, make a GET request to the following endpoint:

```
http://localhost:8080/api/instances?region=your-region&profile=profile-name
```

Replace `your-region` with the AWS region you want to list the instances for and set profile aws config . The response will be a JSON array containing the names, IDs, and states of the instances.

### Resetting a Lightsail Instance

To action a specific Lightsail instance, make a GET request to the following endpoint:

```
http://localhost:8080/api/instance?region=your-region&name=your-instance-name&secret=timestamp&profile=profile-name&action=[reset|changeip|poweroff|poweron]
```

Replace `your-region` with the AWS region and `your-instance-name` with the name of the instance you want to action. The instance action logic should be added to the `resetLightsailInstance` handler in the code.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions to this project are welcome. Please follow the [CONTRIBUTING](CONTRIBUTING.md) guidelines for more details.

## Issues

If you encounter any issues or have questions, please open a GitHub issue in this repository.

## Authors

- [vahiddsy](https://github.com/vahiddsy) - [Your Website](https://antinone.xyz)

## Acknowledgments

- Thanks to the Go community for providing excellent resources and libraries.
- AWS for their powerful Lightsail service.

---