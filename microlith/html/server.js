const express = require('express');
const bodyParser = require('body-parser');
const cors = require('cors');
const fs = require('fs');
const path = require('path');
const http = require('http');

const app = express();
app.use(bodyParser.json());
app.use(cors());

// Get arguments from the command line
const args = process.argv.slice(2); // Skip the first two elements
const configDir = args[0];
const pathPrefix =  "http://localhost:8080";

const configFilePath = path.join(configDir, 'clusters.json');

// Helper function to save clusters to file
function saveClusters(clusters) {
  fs.writeFileSync(configFilePath, JSON.stringify(clusters, null, 2));
}

// Function to send HTTP requests
function sendHttpRequest(options, data) {

  let curlCommand = `curl -X ${options.method} http://${options.hostname}:${options.port}${options.path}`;
  if (data) {
    curlCommand += ` -d '${data}'`;
  }

  if (options.headers) {
    for (const header in options.headers) {
      curlCommand += ` -H '${header}: ${options.headers[header]}'`;
    }
  }

  // Print the request details and the equivalent curl command
  console.info(`Request options: ${JSON.stringify(options, null, 2)}`);
  console.info(`Request data: ${data}`);
  console.info(`Equivalent curl command: ${curlCommand}`);

  return new Promise((resolve, reject) => {
    const req = http.request(options, (res) => {
      let responseData = '';

      res.on('data', (chunk) => {
        responseData += chunk;
      });

      res.on('end', () => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(responseData);
          console.info(JSON.stringify(responseData));
        } else {
          reject(new Error(`Request failed with status code ${res.statusCode}`));
        }
      });
    });

    req.on('error', (error) => {
      console.error(JSON.stringify(error));
      reject(error);
    });

    if (data) {
      req.write(data);
    }

    req.end();
  });
}

// Endpoint to save a new cluster configuration
app.post('/persistency/api/saveConfig', (req, res) => {
  try {
    let clusters = [];
    if (fs.existsSync(configFilePath)) {
      clusters = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));
    }

    const newCluster = req.body;

    // Check if the combination of hostname and managementPort already exists
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

// Endpoint to load all cluster configurations

// Server-side configure function
async function configureCluster(config) 
{
  try {

    await addCluster(config); 

    await addSGW(config);
    await addConfiguration(config);
   
    await refreshPrometheus(config);

    return { success: true, message: 'Cluster successfully configured' };
  } catch (e) {
    return { success: false, message: `Error configuring cluster: ${String(e)}` };
  }

  async function refreshPrometheus(config) {
    var options = {
      method: 'POST',
      path: `${pathPrefix}/prometheus/-/reload`,
      port:'8080',
      headers: {
      }
    };

   await  sendHttpRequest(options, null);
  }

  async function addSGW(config) {
    if (config.doSGW) {
      const postData = JSON.stringify({
        hostname: config.sgwHostname,
        SgwConfig: {
          username: config.sgwUsername,
          password: config.sgwPassword,
        },
        metricsConfig: config.sgwPrometheusPort === null || String(config.sgwPrometheusPort) === "" ? null : {
          metricsPort: parseInt(config.sgwPrometheusPort, 10),
        },
      });

      const options = {

        path: `${pathPrefix}/config/api/v1/sgw/add`,
        method: 'POST',
        port:'8080',
        headers: {
          'Content-Type': 'application/json',

        },
      };

     await  sendHttpRequest(options, postData);
    }
  }


  async function addCluster(config) {


  
    

    
    // Prepare the request data
    const postData = JSON.stringify({
      host: `${config.hostname}:${config.managementPort}`,
      user: config.serverUsername,
      password: config.serverPassword
    });
    
    // Basic Auth Token
    const authToken = 'Basic ' + Buffer.from(`${config.cbmmUsername}:${config.cbmmPassword}`).toString('base64');
    
    // Request options
    const options = {
     
      path: `${pathPrefix}/couchbase/api/v1/clusters`,
      method: 'POST',
      port:'8080',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'Authorization': authToken
      }
    };
    
   await  sendHttpRequest(options, postData);

    
}
  async function addGrafanaDs(config) {






    // Prepare the request data
    const postData = JSON.stringify({
      host: `${config.hostname}:${config.managementPort}`,
      user: config.serverUsername,
      password: config.serverPassword
    });

    // Basic Auth Token
    const authToken = 'Basic ' + Buffer.from(`${config.cbmmUsername}:${config.cbmmPassword}`).toString('base64');

    // Request options
    const options = {

      path: `${pathPrefix}/couchbase/api/v1/clusters`,
      method: 'POST',
      port:'8080',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'Authorization': authToken
      }
    };

   await  sendHttpRequest(options, postData);


}
async function addConfiguration(config) {

   
    
  // Prepare the request data
  const postData = JSON.stringify({
    hostname: config.hostname,
    couchbaseConfig: {
      username: config.serverUsername,
      password: config.serverPassword,
      managementPort: parseInt(config.managementPort, 10),
      useTLS: config.useTLS
    },
    metricsConfig: this.prometheusPort === null || String(this.prometheusPort) === "" ? null : {
      metricsPort: parseInt(this.prometheusPort, 10)
    }
  });
  

  
  // Request options
  const options = {
   
    path: `${pathPrefix}/config/api/v1/clusters/add`,
    method: 'POST',
    port:'8080',
    headers: {
      'Content-Type': 'application/json',

    }
  };
  
  await sendHttpRequest(options, postData);

}

}

// Endpoint to configure a cluster
app.post('/persistency/api/configureCluster', async (req, res) => {
  const config = req.body;
  const result = await configureCluster(config);
  if (result.success) {
    res.status(200).send(result.message);
  } else {
    res.status(500).send(result.message);
  }
});

// Endpoint to load and configure all clusters
app.post('/api/loadAllClusters', async (req, res) => {
  await addAllClusters(res);
});

// Start the server
app.listen(3300, () => {
  console.log('Server is running on port 3300');
   // Delay execution for 2000 ms (2 seconds) and then make the POST request
   setTimeout(async () => {

    addAllClusters(createMockRes()) 
  
  }, 2000);
});
function createMockRes() {
  const res = {};
  res.status = function(statusCode) {
    this.statusCode = statusCode;
    return this;
  };
  res.send = function(body) {
    console.log(`Status: ${this.statusCode}`);
    console.log(`Body: ${body}`);
    return this;
  };
  res.json = function(body) {
    console.log(`Status: ${this.statusCode}`);
    console.log(`JSON Body: ${JSON.stringify(body)}`);
    return this;
  };
  res.statusCode = 200; // Default status code
  return res;
}
async function addAllClusters(res) {
  try {
    if (fs.existsSync(configFilePath)) {
      const clusters = JSON.parse(fs.readFileSync(configFilePath, 'utf8'));

      // Filter out unique clusters based on hostname and managementPort
      const uniqueClusters = [];
      const seen = new Set();

      clusters.forEach(cluster => {
        const identifier = `${cluster.hostname}:${cluster.managementPort}`;
        if (!seen.has(identifier)) {
          seen.add(identifier);
          uniqueClusters.push(cluster);
        }
      });

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

