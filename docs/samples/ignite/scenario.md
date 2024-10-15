# Jumpstart Scenario

In this scenario, you deploy an application with a front-end + back-end web application, a database, and several IoT devices.

## Generic flow for Greenhouse

1. Deploy solution1 and target1 to define the components and deploy method for IoT devices.
    * Deploy a simulate sensor for the temperature and humidity.
    * Will use `providers.target.azure.iotedge`, [deploy method link](https://github.com/eclipse-symphony/symphony/blob/main/docs/symphony-book/get-started/deploy_solution_to_azure_iot_edge.md).
2. Deploy solution2 and target2 for our web application **v1**(front-end + back-end + database).
    * ingress-nginx.
        * Will use `providers.target.helm` to deploy.
    * ingress
        * Will use `providers.target.ingress` to deploy.
    * Front-end and back-end flask application(**v1**).
        * Will use `providers.target.k8s` to deploy.
        * Will have a dashboard to show the temperature history data diagram from the sensor and a button to change the temperature.
        * **TODO**: Write a fancy front-end UI for it.
    * Database. Plan to use mySQL.
        * Will use `providers.target.k8s` to deploy.
        * For storing temperature/humidity data. To avoid data loss when update the web application, we must have one database.
3. Deploy solution3 and target3 for our web application **v2**(front-end + back-end).
    * Front-end and back-end flask application(v2). Will use `providers.target.k8s` to deploy.
        * Will have a dashboard to show the history temperature + humidity data and a button to change the temperature, a button to change the humidity.
4. Deploy three catalogs to describe three instances to map solution and target in step 1, 2 and 3.
5. Deploy campaign1 to deploy IoT device, ingress-nginx + ingress, web application **v1**, and database.
    * **[deploy-IoT]** first stage will materialize the instance describe in the catalog to deploy the IoT devices.
    * **[deploy-application]** second stage will materialize the instance describe in the catalog to help deploy the web application **v1**, database and ingress-nginx + ingress.
    * **[check-web]** third stage will check if a simple http request to the web application is available.
6. Deploy activation1 to trigger campaign1.
7. *The user can use the web application **v1** to see temperature history data diagram, and try to set a target temperature.*
8. Deploy campaign2 to upgrade web application from **v1** to **v2**.
    * **[upgrade]** first stage will materialize the instance describe in the catalog to help deploy the web application **v2**.
    * **[verify]** second stage will use http call to verify the web application **v2** is available.
    * **[switch]** third stage will adjust the ingress weight to move all traffic to web application **v2**.
    * **[destroy-v1]** fourth stage will remove web application **v1**.
9. Deploy activation2 to trigger the campaign2.
10. *The user can reopen the web application and it will have the functionality for humidity (**v2**). Plus: they can refresh the web application during the upgrade operation to see the version switch.*

## Artifacts

| Artifact | Purpose |
|--------|--------|
| activation1.json | Activate the workflow |
| campaign1.json | workflow definition 1 (deploy IoT device, web application, database, ingress-nginx)|
| activation2.json | Activate the workflow 2 |
| campaign2.json | workflow definition 2 (upgrade web application)|
| solution1.json | application definition (IoT devices) |
| target1.json | Target definition for IoT devices |
| solution2.json | Web application **v1** definition (front-end + back-end, database) |
| target2.json | Target definition (current K8s cluster) |
| solution3.json | Web application **v2** definition (front-end + back-end) |
| target3.json | Target definition (current K8s cluster) |
| catalog1.json | describe the 1st target-solution mapping|
| catalog2.json | describe the 2nd target-solution mapping|
| catalog3.json | describe the 3rd target-solution mapping|

Diagram for deployment: **TODO**

Deployment steps: follow the instruction of the private preview docs.
