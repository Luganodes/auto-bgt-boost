<a name="readme-top"></a>
[![Banner](./imgs/banner.png)](https://github.com/Luganodes/Solana-Indexer)

# Auto BGT Boost

Take your validator operations to the next level with this fully automated tool. No more manual intervention required—this tool optimizes your validator's performance by automatically boosting it at regular intervals based on your set thresholds, a process that can be cumbersome and error-prone when done manually. Using Web3Signer, the tool signs and broadcasts transactions seamlessly, ensuring that boosts are queued and activated after 8191 blocks without any effort on your part. It supports multiple validators simultaneously, making it easy to manage and optimize performance across your entire fleet.

<br />
<div align="center">
    <a href="https://github.com/Luganodes/auto-bgt-boost/issues">Report Bug</a>
    |
    <a href="https://github.com/Luganodes/auto-bgt-boost/issues">Request Feature</a>
</div>

## Schema Definitions

### Validator Schema

| Field           | Type   | Description                 |
| --------------- | ------ | --------------------------- |
| Pubkey          | string | Public key of the validator |
| OperatorAddress | string | Address of the operator     |
| BoostThreshold  | string | Threshold for boosting      |

### Activate Boost Schema

| Field           | Type      | Description                                    |
| --------------- | --------- | ---------------------------------------------- |
| Amount          | string    | Amount for boost activation                    |
| ValidatorPubkey | string    | Public key of the validator                    |
| OperatorAddress | string    | Address of the operator                        |
| TransactionHash | string    | Transaction hash                               |
| BlockNumber     | uint64    | Block number in which transaction was included |
| BlockTimestamp  | time.Time | Timestamp of the block                         |
| Fee             | float64   | Transaction fee                                |
| TransactionFrom | string    | Address that initiated the transaction         |
| ToContract      | string    | Contract address receiving the transaction     |

### Queue Boost Schema

| Field           | Type      | Description                                    |
| --------------- | --------- | ---------------------------------------------- |
| ValidatorPubkey | string    | Public key of the validator                    |
| OperatorAddress | string    | Address of the operator                        |
| BlockNumber     | uint64    | Block number in which transaction was included |
| Amount          | string    | Amount for queue boost                         |
| TransactionHash | string    | Transaction hash                               |
| BlockTimestamp  | time.Time | Timestamp of the block                         |
| Fee             | float64   | Transaction fee                                |
| TransactionFrom | string    | Address that initiated the transaction         |
| ToContract      | string    | Contract address receiving the transaction     |

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

- [Go](https://go.dev/doc/install)

- [MongoDB](https://www.mongodb.com/docs/manual/installation/)

- Docker (Optional)
  - For macOS: [Download Docker Desktop for Mac](https://docs.docker.com/desktop/mac/install/)
  - For Windows: [Download Docker Desktop for Windows](https://docs.docker.com/desktop/windows/install/)
  - For Linux: [Docker for Linux](https://docs.docker.com/engine/install/)

### Local Setup

1. Clone the repo

   ```sh
   git clone https://github.com/Luganodes/auto-bgt-boost.git
   cd auto-bgt-boost
   cp .env.sample .env
   ```

2. Popluate .env with appropriate values. Look at [.env.sample](./.env.sample) for reference.

### MakeFile

Build the application

```bash
make build
```

Run the application

```bash
make run
```

Docker run

```bash
make docker-run
```

Shutdown docker containers

```bash
make docker-down
```

Live reload the application

```bash
make watch
```

Clean up binary from the last build

```bash
make clean
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feat/AmazingFeature`)
3. Make some amazing changes.
4. `git add .`
5. Commit your Changes (`git commit -m "<Verb>: <Action>"`)
6. Push to the Branch (`git push origin feat/AmazingFeature`)
7. Open a Pull Request

To start contributing, check out [`CONTRIBUTING.md`](./CONTRIBUTING.md) . New contributors are always welcome to support this project.

## License

Distributed under the MIT License. See [`LICENSE`](./LICENSE) for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>
