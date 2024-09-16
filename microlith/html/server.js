const express = require('express');
const bodyParser = require('body-parser');
const cors = require('cors');
const fs = require('fs');
const path = require('path');
const http = require('http');

const app = express();
app.use(bodyParser.json());
app.use(cors());

const args = process.argv.slice(2); // Get arguments from the command line
const configDir = args[0];
const grafanaadmin=args[1]
const grafanapassword=args[2]
const pathPrefix = "http://localhost:8080";
const configFilePath = path.join(configDir, 'clusters.json');
const apiKeys = new Map();
// Helper function to save clusters to file
function saveClusters(clusters) {
  fs.writeFileSync(configFilePath, JSON.stringify(clusters, null, 2));
}

// Helper function to generate a curl command
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

// Function to send HTTP requests
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

// Helper function to delete an API key by ID
function deleteKey(keyId) {
  const options = {
    method: 'DELETE',
    hostname: 'localhost',
    port: 8080,
    path: `/grafana/api/auth/keys/${keyId}`,
    headers: {
      'Authorization': `Basic ${Buffer.from(grafanaadmin+":"+grafanapassword).toString('base64')}`
    }
  };
  return sendHttpRequest(options).then(() => {
    console.log(`API key with ID ${keyId} deleted successfully.`);
  });
}

// Function to reset (delete all) API keys
async function resetKeys() {
  const options = {
    method: 'GET',
    hostname: 'localhost',
    port: 8080,
    path: '/grafana/api/auth/keys',
    headers: {
      'Authorization': `Basic ${Buffer.from(grafanaadmin+":"+grafanapassword).toString('base64')}`
    }
  };

  const responseBody = await sendHttpRequest(options);
  const keys = JSON.parse(responseBody);

  await Promise.all(keys.map(key => deleteKey(key.id)));
  console.log("All API keys deleted.");
}

// Function to create an API key
async function createApiKey(hostname) {
  if (apiKeys.has(hostname)) {
    console.log(`Key for ${hostname} already exists. Returning from map.`);
    return apiKeys.get(hostname)
  }

  const postData = JSON.stringify({ name: hostname, role: "Admin" });
  const options = {
    method: 'POST',
    hostname: 'localhost',
    port: 8080,
    path: '/grafana/api/auth/keys',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Basic ${Buffer.from(grafanaadmin+":"+grafanapassword).toString('base64')}`
    }
  };

  const responseBody = await sendHttpRequest(options, postData).catch(err => {
    if (err.message.includes('409')) {
      console.error(`Key already exists for ${hostname}. Fetching from map.`);
      return apiKeys.get(hostname);
    }
    throw err;
  });

  if (responseBody) {
    const key = JSON.parse(responseBody).key;
    apiKeys.set(hostname, key);
    console.log(`API key created and stored for ${hostname}`);
    return key;
  }
}

// Function to create a Grafana datasource
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

// Function to refresh Prometheus configuration
async function refreshPrometheus() {
  const options = {
    method: 'POST',
    path: `${pathPrefix}/prometheus/-/reload`,
    port: '8080',
    headers: {}
  };
  await sendHttpRequest(options);
}

// Main function to configure a cluster
async function configureCluster(config) {
  try {
    
    if (!config.hasOwnProperty('grafanaDatasourceTargetPort')){
      config.grafanaDatasourceTargetPort=8093
      upsertCluster(config)
    }
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

// Function to add a Sync Gateway (SGW) configuration
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
      path: `${pathPrefix}/config/api/v1/sgw/add`,
      method: 'POST',
      port: '8080',
      headers: {
        'Content-Type': 'application/json'
      }
    };

    await sendHttpRequest(options, postData);
  }
}

// Function to add a Couchbase cluster
async function addToClusterMonitor(config) {
  const postData = JSON.stringify({
    host: `${config.hostname}:${config.managementPort}`,
    user: config.serverUsername,
    password: config.serverPassword
  });

  const authToken = `Basic ${Buffer.from(`${config.cbmmUsername}:${config.cbmmPassword}`).toString('base64')}`;

  const options = {
    path: `${pathPrefix}/couchbase/api/v1/clusters`,
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

// Function to add a Grafana datasource
async function addGrafanaDs(config) {
  const grafanaKey = await createApiKey(config.alias);
  const dsname = config.alias;
  const datasourceURL = `http://${config.hostname}:${config.grafanaDatasourceTargetPort}`;
  const grafanaURL = `${pathPrefix}/grafana/`;

  const result = await createDatasource(grafanaKey, grafanaURL, dsname, datasourceURL, config.serverUsername, config.serverPassword);
  console.log('Datasource created:', result);
}

// Function to add additional configuration
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
    path: `${pathPrefix}/config/api/v1/clusters/add`,
    method: 'POST',
    port: '8080',
    headers: {
      'Content-Type': 'application/json'
    }
  };

  await sendHttpRequest(options, postData);
}
function upsertCluster(newCluster) {
  let clusters = [];

  // Check if the config file exists and read it
  if (fs.existsSync(configFilePath)) {
    clusters = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));
  }

  // Find the index of the existing cluster
  const index = clusters.findIndex(cluster =>
    cluster.hostname === newCluster.hostname && cluster.managementPort === newCluster.managementPort
  );

  if (index !== -1) {
    // If found, update the existing cluster
    clusters[index] = newCluster;
  } else {
    // If not found, add the new cluster
    clusters.push(newCluster);
  }

  // Save the updated clusters
  saveClusters(clusters);
}
// Endpoint to save a new cluster configuration
app.post('/persistency/api/saveConfig', (req, res) => {
  try {

    // Upsert the cluster configuration
    upsertCluster(req.body);

    res.status(200).send('Cluster configuration saved successfully');
  } catch (error) {
    res.status(500).send('Error saving cluster configuration: ' + error.message);
  }
});

// Endpoint to configure a cluster
app.post('/persistency/api/configureCluster', async (req, res) => {
  const config = req.body;
  const result = await configureCluster(config);
  res.status(result.success ? 200 : 500).send(result.message);
});

// Endpoint to load and configure all clusters
app.post('/api/loadAllClusters', async (req, res) => {
  await addAllClusters(res);
});

// Start the server
app.listen(3300, () => {
  console.log('Server is running on port 3300');
  setTimeout(async () => addAllClusters(createMockRes()), 2000);
});

// Helper to create a mock response object for testing
function createMockRes() {
  const res = {
    statusCode: 200, // Default status code
    status(statusCode) {
      this.statusCode = statusCode;
      return this;
    },
    send(body) {
      console.log(`Status: ${this.statusCode}`);
      console.log(`Body: ${body}`);
      return this;
    },
    json(body) {
      console.log(`Status: ${this.statusCode}`);
      console.log(`JSON Body: ${JSON.stringify(body)}`);
      return this;
    }
  };
  return res;
}

// Function to add all clusters
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
