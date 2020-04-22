#!/usr/bin/env node

const util = require('util');
const exec = util.promisify(require('child_process').exec);
let projects = []; // used to store all projects that will be monitored (along with their dcs, deployments and statefulsets (if any))
let keepRunning = true; // boolean flag used to control the monitoring loop
const start = getCurrentEpochTimestamp();
const NAMESPACE_PREFIX = process.env.NAMESPACE_PREFIX || 'redhat-rhmi-';

monitorDowntime();

async function monitorDowntime() {
  await getProjects(); // get the list of the projects to monitor
  await monitorDowntimePerNs(); // run the loop to monitor the projects until cancelled
  calculateDowntimes(true);
  writeJSONtoFile();
}

/*
Get tdeployment configs using oc command and filter out relevant columns using 'awk'

The unfiltered output looks like this: 

NAMESPACE                       NAME                              REVISION   DESIRED   CURRENT   TRIGGERED BY
...
redhat-rhmi-3scale              apicast-production                1          2         2         config,image(amp-apicast:latest)
redhat-rhmi-3scale              apicast-staging                   1          2         2         config,image(amp-apicast:latest)
redhat-rhmi-3scale              backend-cron                      1          2         2         config,image(amp-backend:latest)
...
*/
async function getDcs() {
  try {
    let dcsOutput = await exec(`oc get dc --all-namespaces | awk '{print $1,$2,$4,$5}'`);
    if (!dcsOutput.stdout.toString().toLocaleLowerCase().includes("no resources") && // else - ns is considered down
      dcsOutput.stdout.toString().length !== 0) {
      let outputLines = dcsOutput.stdout.split("\n"); // split the output to individual lines
      const [_, ...rest] = outputLines; // remove the heading (NAMESPACE NAME ...)
      const dcsLines = rest.filter(e => e !== ''); // remove last empty element
      const dcs = [];
      dcsLines.forEach(line => {
        let splitLines = line.split(/[ ]/); // split lines by space (" ")
        // only deal with the array of 4 elements and desired count not equal to '0'
        if (splitLines.length === 4 && splitLines[0].toString().startsWith(NAMESPACE_PREFIX) && splitLines[2] !== '0') {
          // push new deployment config to the array of dcs
          dcs.push({
            "name": splitLines[1],
            "namespace": splitLines[0],
            "ready": splitLines[3],
            "expected": splitLines[2],
            "downtimes": []
          });
        }
      });
      return dcs;
    } else {
      return [];
    }
  } catch (error) {
    console.log(`Unable to get deployment configs: ${error}`);
    return [];
  }
  
}

/*
Get deployments using oc command and filter out relevant columns using 'awk'

The unfiltered output looks like this: 

NAMESPACE                                               NAME                                                    READY   UP-TO-DATE   AVAILABLE   AGE
...
redhat-rhmi-3scale-operator                             3scale-operator                                         1/1     1            1           17h
redhat-rhmi-amq-online                                  address-space-controller                                1/1     1            1           17h
redhat-rhmi-amq-online                                  api-server                                              1/1     1            1           17h
...
*/
async function getDeployments() {
  try {
    let deploymentsOutput = await exec(`oc get deployment --all-namespaces | awk '{print $1,$2,$3}'`);
    if (!deploymentsOutput.stdout.toString().toLocaleLowerCase().includes("no resources") &&
      deploymentsOutput.stdout.toString().length !== 0) {
      let outputLines = deploymentsOutput.stdout.split("\n"); // split the output to individual lines
      const [_, ...rest] = outputLines; // remove the heading (NAMESPACE NAME)
      const deploymentLines = rest.filter(e => e !== ''); // remove last empty element
      const deployments = [];
      deploymentLines.forEach(line => {
        let splitLines = line.split(/[ ,/]/); // split lines by empty space (" ") and "/"
        // only deal with the array of 4 elements and starting with the namespace prefix
        if (splitLines.length === 4 && splitLines[0].toString().startsWith(NAMESPACE_PREFIX)) {
          deployments.push({
            "name": splitLines[1],
            "namespace": splitLines[0],
            "ready": splitLines[2],
            "expected": splitLines[3],
            "downtimes": [] 
          });
        }
      });
      return deployments;
    } else {
      return [];
    }
  } catch (error) {
    console.log(`Unable to get deployments: ${error}`);
    return [];
  }
}

/*
Get statefulsets using oc command and filter out relevant columns using 'awk'

The unfiltered output looks like this: 

NAMESPACE                                    NAME                                  READY   AGE
...
redhat-rhmi-middleware-monitoring-operator   alertmanager-application-monitoring   1/1     17h
redhat-rhmi-middleware-monitoring-operator   prometheus-application-monitoring     1/1     17h
redhat-rhmi-rhsso                            keycloak                              2/2     17h
redhat-rhmi-user-sso                         keycloak                              2/2     17h
...
*/
async function getStatefulSets() {
  try {
    let statefulSetsOutput = await exec(`oc get statefulset --all-namespaces | awk '{print $1,$2,$3}'`);
    if (!statefulSetsOutput.stdout.toString().toLocaleLowerCase().includes("no resources") &&
    statefulSetsOutput.stdout.toString().length !== 0) {
      let outputLines = statefulSetsOutput.stdout.split("\n"); // split the output to individual lines
      const [_, ...rest] = outputLines; // remove the heading (NAMESPACE NAME)
      const statefulSetLines = rest.filter(e => e !== ''); // remove last empty element
      const statefulSets = [];
      statefulSetLines.forEach(line => {
        let splitLines = line.split(/[ ,/]/); // split lines by empty space (" ") and "/"
        // only deal with the array of 4 elements and starting with the namespace prefix
        if (splitLines.length === 4 && splitLines[0].toString().startsWith(NAMESPACE_PREFIX)) {
          statefulSets.push({
            "name": splitLines[1],
            "namespace": splitLines[0],
            "ready": splitLines[2],
            "expected": splitLines[3],
            "downtimes": []
          });
        }
      });
      return statefulSets;
    } else {
      return [];
    }
  } catch (error) {
    console.log(`Unable to get deployments: ${error}`);
    return [];
  }
}

/**
 * 
 * @param {*} completeDowntimes - whether to set all downtimes with unset end timestamp (0) to current timestamp - 
 * only at the end of the script execution
 */
function calculateDowntimes(completeDowntimes) {
  for (let projIndex = 0; projIndex < projects.length; projIndex++) {
    if (projects[projIndex].downtimes.length !== 0) {
      if (projects[projIndex].downtimes[projects[projIndex].downtimes.length - 1].end === 0) { // if there are some downtimes already and the last downtime does not have 'end' timestamp
        if (completeDowntimes) {
          projects[projIndex].downtimes[projects[projIndex].downtimes.length - 1].end = getCurrentEpochTimestamp();
        }
      }
      projects[projIndex].downtimeInSeconds = getTotalDowntime(projects[projIndex].downtimes);
    }
  }
}

/*
Write current projects object to JSON file
*/
function writeJSONtoFile() {
  process.stdout.write('\nPersisting current JSON data to downtime.json file... ');
  const fs = require('fs');
  const results = {"projects": projects, "start": start, "end": getCurrentEpochTimestamp()};
  fs.writeFile("downtime.json", JSON.stringify(results), function(err) {
    if (err) {
      console.log(err);
    }
  });
}

/*
Calculate total downtime (those ending with '0' will be ommited)
*/
function getTotalDowntime(downtimes) {
  let downtimeInSeconds = 0;
  try {
    downtimes.forEach(downtime => {
      if (downtime.end !== 0) {
        downtimeInSeconds += (downtime.end - downtime.start);
      }
    });
    return downtimeInSeconds;
  } catch (_) {
    return 0;
  }
}

/**
 * 
 * @param {*} item - item to be found
 * @param {*} array - array to be traversed
 * 
 * returns index of the item in the specified array
 */
function findByNameAndNamespace(item, array) {
  try {
    return array.find(
      element => 
        element.name === item.name &&
        element.namespace === item.namespace
      );
  } catch (error) {
    console.log(`Unable to find an item: ${array}` );
    return undefined;
  }
}

/**
 * 
 * @param {*} isDown - whether namespace or resource is down
 * @param {*} namespaceIndex - index of the namespace in the projects array
 * @param {*} resourceIndex - index of the resource 
 * @param {*} resource - specifies whether to update namespace or resource
 * @param {*} timestamp - timestamp to set
 * @param {*} downtimeIndex - which of the objects from downtime array to update
 */
function updateDowntime(isDown, namespaceIndex, resourceIndex, resource, timestamp, downtimeIndex) {
  if (isDown) {
    if (keepRunning) {
      switch(resource) {
        case "dc":
          projects[namespaceIndex].dcs[resourceIndex].downtimes.push({"start": timestamp, "end": 0});
          break;
        case "deployment":
          projects[namespaceIndex].deployments[resourceIndex].downtimes.push({"start": timestamp, "end": 0});
          break;
        case "statefulset":
          projects[namespaceIndex].statefulsets[resourceIndex].downtimes.push({"start": timestamp, "end": 0});
          break;
        case "namespace":
          projects[namespaceIndex].downtimes.push({"start": timestamp, "end": 0});
          break;
      }
    }
  } else {
    switch(resource) {
      case "dc":
        projects[namespaceIndex].dcs[resourceIndex].downtimes[downtimeIndex].end = timestamp;
        break;
      case "deployment":
        projects[namespaceIndex].deployments[resourceIndex].downtimes[downtimeIndex].end = timestamp
        break;
      case "statefulset":
        projects[namespaceIndex].statefulsets[resourceIndex].downtimes[downtimeIndex].end = timestamp
        break;
      case "namespace":
        projects[namespaceIndex].downtimes[downtimeIndex].end = timestamp;
        break;
    }
  }
}

/*
Continuously run the loop and monitor the resources per namespace until 'keepRunning' is set to false
*/
async function monitorDowntimePerNs() {
  while (keepRunning) {
    try {
      process.stdout.write(`\nGetting available deployments, dcs and statefulsets... `);
      const currentDcs = await getDcs();
      const currentDeployments = await getDeployments();
      const currentStatefulSets = await getStatefulSets();
      process.stdout.write(`done`);
      const timestamp = getCurrentEpochTimestamp();
      for (let projIndex = 0; projIndex < projects.length; projIndex++) {
        let isNamespaceReady = true; // unless set to false namespace is considered ready
        // checking dcs of the namespace (if any)
        if (projects[projIndex].dcs.length > 0) {
          process.stdout.write(`\nChecking readiness of deployment configs in ${projects[projIndex].name} namespace...`);
          for (let dcIndex = 0; dcIndex < projects[projIndex].dcs.length; dcIndex++) {
            const dc = findByNameAndNamespace(projects[projIndex].dcs[dcIndex], currentDcs);
            const downtimesSize = projects[projIndex].dcs[dcIndex].downtimes.length;
            if (dc === undefined || dc.ready === "0") { // if no expected dc or dc ready === 0
              isNamespaceReady = false;
              if (downtimesSize === 0 || projects[projIndex].dcs[dcIndex].downtimes[downtimesSize - 1].end !== 0) {
                updateDowntime(true, projIndex, dcIndex, "dc", timestamp, downtimesSize);
              }
            } else { // set downtime.end if not set
              if (downtimesSize !== 0 && projects[projIndex].dcs[dcIndex].downtimes[downtimesSize - 1].end === 0) {
                updateDowntime(false, projIndex, dcIndex, "dc", timestamp, downtimesSize - 1);
              }
            }
          }
        }
        // checking deployments of the namespace (if any)
        if (projects[projIndex].deployments.length > 0) {
          process.stdout.write(`\nChecking readiness of deployments in ${projects[projIndex].name} namespace...`);
          for (let dIndex = 0; dIndex < projects[projIndex].deployments.length; dIndex++) {
            const deployment = findByNameAndNamespace(projects[projIndex].deployments[dIndex], currentDeployments);
            const downtimesSize = projects[projIndex].deployments[dIndex].downtimes.length;
            if (deployment === undefined || deployment.ready === "0") { // if no expected deployment or deployment ready === 0
              isNamespaceReady = false;
              if (downtimesSize === 0 || projects[projIndex].deployments[dIndex].downtimes[downtimesSize - 1].end !== 0) {
                updateDowntime(true, projIndex, dIndex, "deployment", timestamp, downtimesSize);
              }
            } else { // set downtime.end if not set
              if (downtimesSize !== 0 && projects[projIndex].deployments[dIndex].downtimes[downtimesSize - 1].end === 0) {
                updateDowntime(false, projIndex, dIndex, "deployment", timestamp, downtimesSize - 1);
              }
            }
          }
        }
        // checking statefulsets of the namespace (if any)
        if (projects[projIndex].statefulsets.length > 0) {
          process.stdout.write(`\nChecking readiness of statefulsets in ${projects[projIndex].name} namespace...`);
          for (let sIndex = 0; sIndex < projects[projIndex].statefulsets.length; sIndex++) {
            const statefulSet = findByNameAndNamespace(projects[projIndex].statefulsets[sIndex], currentStatefulSets);
            const downtimesSize = projects[projIndex].statefulsets[sIndex].downtimes.length;
            if (statefulSet === undefined || statefulSet.ready === "0") { // if no expected statefulset or statefulset ready === 0
              isNamespaceReady = false;
              if (downtimesSize === 0 || projects[projIndex].statefulsets[sIndex].downtimes[downtimesSize - 1].end !== 0) {
                updateDowntime(true, projIndex, sIndex, "statefulset", timestamp, downtimesSize);
              }
            } else { // set downtime.end if not set
              if (downtimesSize !== 0 && projects[projIndex].statefulsets[sIndex].downtimes[downtimesSize - 1].end === 0) {
                updateDowntime(false, projIndex, sIndex, "statefulset", timestamp, downtimesSize - 1);
              }
            }
          }
        }
        const downtimesSize = projects[projIndex].downtimes.length;
        if (isNamespaceReady) { // set downtime.end if not set
          if (downtimesSize > 0 && projects[projIndex].downtimes[downtimesSize - 1].end === 0) {
            updateDowntime(false, projIndex, -1, "namespace", timestamp, downtimesSize - 1);
          }
        } else { // if any of the monitored ns resources are not ready - the ns is also not ready
          if (downtimesSize === 0 || projects[projIndex].downtimes[downtimesSize - 1].end !== 0) {
            updateDowntime(true, projIndex, -1, "namespace", timestamp, downtimesSize - 1);
          }
        }
      }
    } catch (error) {
      console.log(error);
    }
    if (keepRunning) {
      calculateDowntimes(false); // update downtimes 
      writeJSONtoFile(); // write to file on each loop
    }
  }
}

/*
Get all namespaces starting with the specified (or default) namespace prefix
*/
async function getProjects() {
  console.log("Getting initial list of RHMI components");
  projects = [];
  const ocResponse = await exec(`oc get projects -o json | jq '.items[] | select(.metadata.name | startswith(\"${NAMESPACE_PREFIX}\")) |.metadata.name'`);
  let projectNames = ocResponse.stdout.split(/\r?\n/).filter(e => e !== '');
  const dcs = await getDcs();
  const deployments = await getDeployments();
  const statefulsets = await getStatefulSets();
  projectNames.forEach(project => {
    const namespaceName = (project.startsWith('"') && project.endsWith('"')) ? project.slice(1, -1) : project; // remove the quotes (if present)
    projects.push({
      "name": namespaceName,
      "dcs": dcs.filter(dc => dc.namespace === namespaceName),
      "deployments": deployments.filter(d => d.namespace === namespaceName),
      "statefulsets": statefulsets.filter(s => s.namespace === namespaceName),
      "downtimes": [],
      "downtimeInSeconds" : 0});
  });
}

/*
Get current timestamp (in seconds)
*/
function getCurrentEpochTimestamp() {
  return Math.floor(Date.now() / 1000);
}

/*
Catch 'Ctrl + C and set 'keepRunning' to false 
*/
process.on('SIGINT', async function() {
  console.log("Caught interrupt signal");
  keepRunning = false;
  calculateDowntimes(true);
  await exec('sleep 10');
  process.exit(0);
});
