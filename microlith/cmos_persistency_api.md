



![image-20240902223201957](/Users/josecortesdiaz/Library/Application Support/typora-user-images/image-20240902223201957.png)



[toc]



# CMOS enhanced Functionality Executive Summary. 

This document covers the canhes performed to the couchbase [CMOS stack](https://docs.couchbase.com/cmos/current/index.html)

The code changes have been commited to [couchbaselabs](https://github.com/couchbaselabs/observability/tree/bbva)

The code has been handled to [BBVA](https://www.bbva.com/) for management of their production Clusters.

## Current State CMOS

![image-20240902215231807](/Users/josecortesdiaz/Library/Application Support/typora-user-images/image-20240902215231807.png)



The diagrams represent the architecture of a monitoring system that integrates with multiple clusters to collect metrics and manage configurations.  The system is deployed as a Docker container.

The operation  "Add cluster" allows clusters to be added to the CMOS monitoring system.

The api is responsible for adding clusters to the monitoring system **Cluster Monitor**.

It also performs the **Prometheus Integration**. The configuration of Prometheus (including cluster details) is managed through a configuration file. That file is modified by the add cluster api and allows to add clusters dynamically. 



## Updated CMOS

![image-20240902220304551](/Users/josecortesdiaz/Library/Application Support/typora-user-images/image-20240902220304551.png)

In the old architecture, storing metrics in the same container creates a single point of failure. If the container fails, all metrics are lost, and the system will miss crucial information about the failure. This setup not only results in losing metrics but also resets configurations, requiring an admin to manually re-add all clusters. This process is tedious and, more importantly, contradicts the principle of system continuity. If a failure occurs when no admin is available, there will be periods during which the system contains no metrics.

The new architecture improves on this by **saving cluster and datasource configurations**. These configurations persist through container restarts or system failures, significantly enhancing data durability and recovery. The slow queries dashboard requires a datasource linked to a Dataservice node, which can be queried via REST. This setup is complex because CMOS initially only supported a metrics datasource (Prometheus). The new architecture addresses this by creating a datasource for each cluster using the JSON REST plugin, ensuring that each cluster has its own unique datasource that is also persisted alongside configurations.

All these improvements are achieved with the addition of a **Docker volume**. The new architecture uses this Docker volume for storing configurations and metrics, enhancing both the persistence and reliability of stored data. With the addition of configuration and metrics storage, along with new data sources, the API’s scope has expanded, allowing it to handle more complex data and provide deeper insights into cluster performance. To manage these advanced operations, a new **persistence layer** has been added to the system. This layer, an additional API deployed with the CMOS monolith, requires integration into the Nginx reverse proxy.

![image-20240902221731943](/Users/josecortesdiaz/Library/Application Support/typora-user-images/image-20240902221731943.png)

## CMOS Persistency API 

This section provides a detailed explanation of an Express.js API that manages configurations for clusters, API keys, and datasources. The API uses various utility functions and endpoints to interact with a server, handling tasks such as HTTP requests, file management, and communication with Grafana and Couchbase.

## Prerequisites

- **Node.js** and **npm**: Make sure you have Node.js and npm installed on your system to run this Express application.

## Dependencies

- **express**: A web framework for Node.js to handle HTTP requests and responses.
- **body-parser**: Middleware to parse incoming request bodies in a middleware before your handlers, available under the `req.body` property.
- **cors**: Middleware to enable CORS (Cross-Origin Resource Sharing) for the API.
- **fs**: Node.js module for interacting with the file system, used to read and write configuration files.
- **path**: Node.js module to handle and transform file paths.
- **http**: Node.js module to make HTTP requests, used for internal server communication.

## Initial Setup

**Express Application Setup**:

```javascript
const app = express();
app.use(bodyParser.json());
app.use(cors());
```
Initializes an Express application and uses `body-parser` and `cors` middleware to handle JSON requests and allow cross-origin requests.

**Command Line Arguments**:

```javascript
const args = process.argv.slice(2);
const configDir = args[0];
const grafanaadmin=args[1]
const grafanapassword=args[2]
const pathPrefix = "http://localhost:8080";
const configFilePath = path.join(configDir, 'clusters.json');
const apiKeys = new Map();
```
Retrieves command-line arguments, specifically the configuration directory, which is expected to contain the `clusters.json` file.



## HTTP REST Endpoints

### `Save Configuration Endpoint`

```javascript
app.post('/persistency/api/saveConfig', (req, res) => {
  try {
    let clusters = [];
    if (fs.existsSync(configFilePath)) {
      clusters = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));
    }

    const newCluster = req.body;
    const exists = clusters.some(cluster => cluster.hostname === newCluster.hostname && cluster.managementPort === newCluster.managementPort);

    if (exists) {
      res.status(400).send('Cluster with the same hostname and management port already exists');
    } else {
      clusters.push(newCluster);
      saveClusters(clusters);
      res.status(200).send('Cluster configuration saved successfully');
    }
  } catch (error) {
    res.status(500).send('Error saving cluster configuration: ' + error.message);
  }
});


```


Saves a new cluster configuration received in the request body. Checks for duplicates before saving.



#### `SaveCluster`

```javascript
function saveClusters(clusters) {
  fs.writeFileSync(configFilePath, JSON.stringify(clusters, null, 2));
}
```

Writes the new cluster configuration received to a configuration file. 



### `Configure Cluster Endpoint`

```javascript
app.post('/persistency/api/configureCluster', async (req, res) => {
 const config = req.body;
  const result = await configureCluster(config);
  res.status(result.success ? 200 : 500).send(result.message);
});
```
Configures a cluster based on the configuration data received in the request body.

#### `configureCluster`

```javascript
async function configureCluster(config) {
  try {
    await resetKeys();
    await addGrafanaDs(config);
    await addToClusterMonitor(config);
    await addSGW(config);
    await addToPrometheus(config);
    await refreshPrometheus(config);
    return { success: true, message: 'Cluster successfully configured' };
  } catch (e) {
    return { success: false, message: `Error configuring cluster: ${e.message}` };
  }
}
}
```

Configures a cluster by calling various functions such as resetting keys, adding datasources, and configuring Couchbase clusters and Sync Gateway.

#### `refreshPrometheus`

```javascript
async function refreshPrometheus() {
 const options = {
    method: 'POST',
    path: `${pathPrefix}/prometheus/-/reload`,
    port: '8080',
    headers: {}
  };
  await sendHttpRequest(options);
}
```

Sends a POST request to reload Prometheus configuration.



#### Configuration functions

These functions are for management of the Sync Gateway, Couchbase clusters and Grafana datasources,  They use HTTP requests to interact with various services, handling configuration data and managing resources.

Ensure that the configuration object (`config`) passed to each function contains the necessary fields and values to avoid errors during execution.

##### `addSGW(config)`

The `addSGW` function adds a Sync Gateway (SGW) configuration based on the provided `config` object. This function checks if the SGW configuration should be added (`config.doSGW`), then sends an HTTP POST request to add the configuration to the server.

```javascript
async function addSGW(config) {
  if (config.doSGW) {
    const postData = JSON.stringify({
      hostname: config.sgwHostname,
      SgwConfig: {
        username: config.sgwUsername,
        password: config.sgwPassword,
      },
      metricsConfig: config.sgwPrometheusPort ? { metricsPort: parseInt(config.sgwPrometheusPort, 10) } : null
    });

    const options = {
      path: \`\${pathPrefix}/config/api/v1/sgw/add\\`,
      method: 'POST',
      port: '8080',
      headers: {
        'Content-Type': 'application/json'
      }
    };

    await sendHttpRequest(options, postData);
  }
}
```

- **`config`**: Object containing configuration details such as hostname, username, password, and metrics configuration.
- **`postData`**: JSON stringified data that includes the hostname, SGW configuration (username and password), and optional metrics configuration.
- **`options`**: HTTP request options for the POST request, specifying the path, method, port, and headers.
- **`sendHttpRequest`**: Function that sends the HTTP request with the specified options and data.

##### `addToClusterMonitor(config)`

The `addToClusterMonitor` function adds a Couchbase cluster configuration using the provided `config` object. It sends an HTTP POST request to add the cluster details, handling any errors related to unique constraints. This is for **adding the cluster in the cluster monitor UI.**

```javascript
async function addToClusterMonitor(config) {
  const postData = JSON.stringify({
    host: \`\${config.hostname}:\${config.managementPort}\\`,
    user: config.serverUsername,
    password: config.serverPassword
  });

  const authToken = \`Basic \${Buffer.from(\`\${config.cbmmUsername}:\${config.cbmmPassword}\`).toString('base64')}\\`;

  const options = {
    path: \`\${pathPrefix}/couchbase/api/v1/clusters\\`,
    method: 'POST',
    port: '8080',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': authToken
    }
  };

  const responseBody = await sendHttpRequest(options, postData).catch(err => {
    if (err.message.includes('{"status":500,"msg":"could not save cluster","extras":"could not add cluster: UNIQUE constraint failed: clusters.uuid"}')) {
      return `{"message": "Cluster exists"}`;
    }
    throw err;
  });

  console.info(responseBody);
}
```

- **`config`**: Object containing the cluster's hostname, management port, server username, server password, Couchbase Management Manager (CBMM) username, and password.
- **`postData`**: JSON stringified data with the cluster's host, username, and password.
- **`authToken`**: Authorization token created by encoding the CBMM username and password in Base64.
- **`options`**: HTTP request options for the POST request, specifying the path, method, port, headers, and authorization token.
- **`sendHttpRequest`**: Function that sends the HTTP request with the specified options and data.
- **Error Handling**: Checks for a specific error message indicating that the cluster already exists and handles it gracefully.

##### `addToPrometheus(config)`

The `addConfiguration` function adds additional configurations for a cluster using the provided `config` object. It sends an HTTP POST request to add the configuration details, including metrics if specified. This is for **adding the configuration of the cluster in the prometheus config file**.  Remember prometheus pulls the metrics from the list of configured clusters. It requires restart prometheus to make the changes effective

```javascript
async function addToPrometheus(config) {
  const postData = JSON.stringify({
    hostname: config.hostname,
    couchbaseConfig: {
      username: config.serverUsername,
      password: config.serverPassword,
      managementPort: parseInt(config.managementPort, 10),
      useTLS: config.useTLS
    },
    metricsConfig: config.prometheusPort ? { metricsPort: parseInt(config.prometheusPort, 10) } : null
  });

  const options = {
    path: \`\${pathPrefix}/config/api/v1/clusters/add\\`,
    method: 'POST',
    port: '8080',
    headers: {
      'Content-Type': 'application/json'
    }
  };

  await sendHttpRequest(options, postData);
}
```

- **`config`**: Object containing the hostname, server username, server password, management port, TLS usage, and optional Prometheus port for metrics.
- **`postData`**: JSON stringified data including hostname, Couchbase configuration, and optional metrics configuration.
- **`options`**: HTTP request options for the POST request, specifying the path, method, port, and headers.
- **`sendHttpRequest`**: Function that sends the HTTP request with the specified options and data.

##### `addGrafanaDs(config)`

The `addGrafanaDs` function adds a Grafana datasource using the provided `config` object. It creates a new Grafana API key and datasource, using the specified server details. This data source is needed in order to setup the slow queries dashboard, its a JSON datasource that **connects to the query services via the REST interface**

```javascript
async function addGrafanaDs(config) {
  const grafanaKey = await createApiKey(config.alias);
  const dsname = config.alias;
  const datasourceURL = \`http://\${config.hostname}:8093\\`;
  const grafanaURL = \`\${pathPrefix}/grafana/\\`;

  const result = await createDatasource(grafanaKey, grafanaURL, dsname, datasourceURL, config.serverUsername, config.serverPassword);
  console.log('Datasource created:', result);
}
```

- **`config`**: Object containing the alias, hostname, server username, and server password.
- **`grafanaKey`**: API key created for Grafana using the `createApiKey` function.
- **`dsname`**: Datasource name, set to the alias provided in the config.
- **`datasourceURL`**: URL for the datasource, constructed using the hostname and port.
- **`grafanaURL`**: Base URL for Grafana.
- **`createDatasource`**: Function that creates the datasource in Grafana using the provided parameters.
- **`console.log`**: Logs the result of the datasource creation.



##### `createDatasource`

```javascript
async function createDatasource(grafanaKey, grafanaURL, dsname, datasourceURL, basicAuthUser, basicAuthPassword) {
   const datasourceConfig = {
    name: dsname,
    type: 'marcusolsson-json-datasource',
    typeName: 'JSON API',
    typeLogoUrl: 'public/plugins/marcusolsson-json-datasource/img/logo.svg',
    access: 'proxy',
    url: datasourceURL,
    basicAuth: true,
    basicAuthUser,
    basicAuthPassword,
    isDefault: false,
    jsonData: {},
    readOnly: false
  };

  const options = {
    hostname: new URL(grafanaURL).hostname,
    port: new URL(grafanaURL).port || 8080,
    path: '/grafana/api/datasources',
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${grafanaKey}`
    }
  };

  const responseBody = await sendHttpRequest(options, JSON.stringify(datasourceConfig)).catch(err => {
    if (err.message.includes('409')) {
      return `{"message": "Datasource already exists for ${dsname}"}`;
      
    }
    throw err;
  });
  return JSON.parse(responseBody);
}
```

Creates a new datasource in Grafana using the specified parameters and a provided Grafana API key.

### `Load All Clusters Endpoint`

```javascript
app.post('/api/loadAllClusters', async (req, res) => {
    await addAllClusters(res);
});


```
Loads and configures all clusters from the `clusters.json` file.

#### `addAllClusters`

```javascript
async function addAllClusters(res) {
  try {
    if (fs.existsSync(configFilePath)) {
      const clusters = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));

      const uniqueClusters = [...new Map(clusters.map(cluster => [`${cluster.hostname}:${cluster.managementPort}`, cluster])).values()];

      const results = await Promise.all(uniqueClusters.map(configureCluster));
      const failedConfigs = results.filter(result => !result.success);

      if (failedConfigs.length > 0) {
        res.status(500).send(`Some clusters failed to configure: ${failedConfigs.map(f => f.message).join(', ')}`);
      } else {
        res.status(200).send('All clusters configured successfully');
      }
    } else {
      res.status(404).send('Configuration file not found');
    }
  } catch (error) {
    res.status(500).send('Error loading and configuring clusters: ' + error.message);
  }
}
```



Reads the configuration file and restores the cmos metrics  and cluster objects.

## Server Initialization

The server listens on port `3300` and automatically attempts to load all clusters shortly after starting.

```javascript
app.listen(3300, () => {
  console.log('Server is running on port 3300');
  setTimeout(async () => addAllClusters(createMockRes()), 2000);
});
```

It loads all clusters every time the server starts. This is triggered by the entrypoints in the CMOS monolith everytime cmos starts. 

## Utility Functions



### `generateCurlCommand`

```javascript
function generateCurlCommand(options, data) {
  let curlCommand = `curl -X ${options.method} http://${options.hostname}:${options.port}${options.path}`;
  if (data) curlCommand += ` -d '${data}'`;
  if (options.headers) {
    for (const [header, value] of Object.entries(options.headers)) {
      curlCommand += ` -H '${header}: ${value}'`;
    }
  }
  return curlCommand;
}
```

Generates a `curl` command equivalent to the HTTP request being made. Useful for debugging and logging.

### `sendHttpRequest`

```javascript
function sendHttpRequest(options, data = null) {
  const curlCommand = generateCurlCommand(options, data);

  console.info(`Request options: ${JSON.stringify(options, null, 2)}`);
  console.info(`Request data: ${data}`);
  console.info(`Equivalent curl command: ${curlCommand}`);

  return new Promise((resolve, reject) => {
    const req = http.request(options, (res) => {
      let responseData = '';

      res.on('data', (chunk) =>{
         responseData += chunk;
        
        } );
      res.on('end', () => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(responseData);
        } else {
          reject(new Error(`Request failed with status code ${res.statusCode} ${responseData}`));
        }
      });
    });

    req.on('error', reject);
    if (data) req.write(data);
    req.end();
  });
}
```

Sends HTTP requests using the Node.js `http` module. Logs the request details and handles responses, resolving or rejecting a Promise based on the HTTP status code.

## API Key Management Functions

### `deleteKey`

```javascript
function deleteKey(keyId) {
   const options = {
    method: 'DELETE',
    hostname: 'localhost',
    port: 8080,
    path: `/grafana/api/auth/keys/${keyId}`,
    headers: {
      'Authorization': `Basic ${Buffer.from(grafanaadmin:grafanapassword).toString('base64')}`
    }
  };
  return sendHttpRequest(options).then(() => {
    console.log(`API key with ID ${keyId} deleted successfully.`);
  });
}
```

Deletes an API key from Grafana using its ID. Makes a DELETE HTTP request to the Grafana API.

### `resetKeys`

```javascript
async function resetKeys() {
   const options = {
    method: 'GET',
    hostname: 'localhost',
    port: 8080,
    path: '/grafana/api/auth/keys',
    headers: {
      'Authorization': `Basic ${Buffer.from(grafanaadmin:grafanapassword).toString('base64')}`
    }
  };

  const responseBody = await sendHttpRequest(options);
  const keys = JSON.parse(responseBody);

  await Promise.all(keys.map(key => deleteKey(key.id)));
  console.log("All API keys deleted.");
}
```

Resets (deletes) all existing API keys in Grafana. First retrieves all keys and then deletes each one.

### `createApiKey`

```javascript
async function createApiKey(hostname) {
   if (apiKeys.has(hostname)) {
    console.log(`Key for ${hostname} already exists. Returning from map.`);
    return apiKeys.get(hostname);
  }

  const postData = JSON.stringify({ name: hostname, role: "Admin" });
  const options = {
    method: 'POST',
    hostname: 'localhost',
    port: 8080,
    path: '/grafana/api/auth/keys',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Basic ${Buffer.from(grafanaadmin:grafanapassword).toString('base64')}`
    }
}
```

Creates a new API key in Grafana for a specific hostname. Checks if a key already exists in the `apiKeys` map before creating a new one.
