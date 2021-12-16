/*
# ---------------------------------------------------------------------------------
#   GENERAL CONFIGURATION
# ---------------------------------------------------------------------------------
- Variable: FORCE_DEPLOY_INTEGRATION
  Description: When true, all artifacts will be published and deployed to the test environment, no matter the branch
  Type: Bool
  Stages:
    - Push docker image to Harbor
    - Push helm chart to Harbor
    - Deploy to test (This depends on ARGODEPLOY variable too)
*/
FORCE_DEPLOY_INTEGRATION = true

/*
# ---------------------------------------------------------------------------------
#   DOCKER CONFIGURATION
# ---------------------------------------------------------------------------------
- Variable: IMAGES
  Description: Dockerfiles used, define where those are located, and the name of the image that should be used to build each Dockerfile
  Type: [[]String]
        name: (String) Name of the image that will be built, it must match Harbor's path
        path: (String) Relative path of the Dockerfile inside this repository. If Dockerfile is at the root, just type a dot '.'
        additionalNames: ([]String) (Optional) Indicate additional names, that will be used as extra tags of the image
  Stages:
    - Building Docker Image
    - Pushing Docker Image For Develop
    - Pushing Docker Image
*/

IMAGES = [
  [name: 'platform/cost-report', path: '.']
]

/*
# ---------------------------------------------------------------------------------
#   ARGOCD CONFIGURATION
# ---------------------------------------------------------------------------------
- Variable: ARGOREPO
  Description: Name of the GitHub repository where ArgoCD listens for changes
  Type: String
  Stages:
    - N/A
  Functions:
    - callArgo
- Variable: ARGOBRANCH
  Description: Branch of the GitHub repository where ArgoCD listens for changes
  Type: String
  Stages:
    - N/A
  Functions:
    - callArgo
- Variable: ARGODEPLOY
  Description: Whether or not update ArgoCD's GitHub repository automatically
  Type: Bool
  Stages:
    - Update ArgoCD deployment
- Variable: ARGOFOLDERS
  Description: List of folders inside ArgoCD's GitHub repository that will be updated
  Type: []String
  Stages:
    - N/A
  Functions:
    - callArgo
*/
ARGOREPO_TEST = 'platform-eks-test-workloads'
ARGOREPO_STAGE = 'platform-eks-staging-workloads'
ARGOREPO_PROD = 'platform-eks-production-workloads'

ARGOBRANCH = 'main'
ARGODEPLOY = false
ARGOFOLDERS = './applications/platform/monitoring/cost-report'

/*
# ---------------------------------------------------------------------------------
#   HELM CONFIGURATION
# ---------------------------------------------------------------------------------
- Variable: CHART_PATH
  Description: Relative path of the Helm Chart inside this repository
  Type: String
  Stages:
    - Linting Helm Template
    - Helm Push
- Variable: HELM_REPO
  Description: Helm Repository URL to push the Chart
  Type: String
  Stages:
    - Linting Helm Template
    - Helm Push
*/
CHART_PATH = 'helm'
HELM_REPO = 'https://harbor.internal.shared.empathy.co/chartrepo/empathyco'

pipeline {
    agent { label 'docker' }
    options {
        buildDiscarder(logRotator(numToKeepStr: '30', artifactNumToKeepStr: '10'))
    }

    environment {
        DOCKER_BUILDKIT = 0
    }

    stages {

        stage('Preparation') {
            stages {
                stage('Set global vars') {
                    steps {
                        script {
                            REVISION = sh(script: 'git describe --tags --always', returnStdout: true).trim()
                            echo "Revision (git describe): ${REVISION}"
                        }
                    }
                }

                stage('Init release') {
                    when {
                        buildingTag()
                    }
                    steps {
                        script {
                            RELEASE = true
                        }
                    }
                }
            }
        }

        stage('Build Docker Image') {
            steps {
                dockerBuild(IMAGES)
            }
        }

        stage('Push Docker Image') {
            environment {
                REGISTRY = 'harbor.internal.shared.empathy.co'
                REGISTRY_CREDENTIALS = 'harbor-credentials'
            }
            when {
                anyOf {
                    branch 'main'
                    expression { FORCE_DEPLOY_INTEGRATION }
                    buildingTag()
                }
                beforeAgent true
            }
            steps {
                dockerPush(IMAGES, REVISION)
            }
        }

        stage('Helm') {
             agent {
                 docker {
                   image 'devth/helm:v3.5.2'
                   args "-u 0:0"
                }
             }

             stages{
                stage('Lint Template') {

                     steps {
                         script {
                             sh "helm lint ${CHART_PATH} -f ${CHART_PATH}/values.yaml"
                         }
                     }
                }

                stage ('Push Helm Chart') {
                    when {
                        anyOf {
                            branch 'main'
                            expression { FORCE_DEPLOY_INTEGRATION }
                            buildingTag()
                        }
                        beforeAgent true
                    }

                    steps {
                          sh "helm plugin install https://github.com/chartmuseum/helm-push"
                          withCredentials([usernamePassword(credentialsId: 'harbor-credentials', usernameVariable: 'HELM_REPO_USERNAME', passwordVariable: 'HELM_REPO_PASSWORD')]) {
                              sh "helm cm-push ${CHART_PATH} ${HELM_REPO} -v ${REVISION}"
                          }
                    }
                }
             }
        }

        stage('Deploy with ArgoCD') {

            agent {
                docker {
                   image 'devth/helm:v3.5.2'
                   args "-u 0:0"
                   reuseNode true
                }
            }
            stages {
                stage('Update ArgoCD test') {
                    when {
                        anyOf {
                            branch 'main'
                            expression { FORCE_DEPLOY_INTEGRATION }
                        }
                        expression { ARGODEPLOY }
                        beforeAgent true
                    }

                    steps {
                        updateArgoRepository(ARGOFOLDERS, REVISION, "main", ARGOREPO_TEST)
                    }
                }
                stage('Update ArgoCD staging') {
                    when {
                        allOf {
                            buildingTag()
                            expression { isTaggedCommitOnMainBranch() }
                        }
                        expression { ARGODEPLOY }
                        beforeAgent true
                    }

                    steps {
                        updateArgoRepository(ARGOFOLDERS, REVISION, "main", ARGOREPO_STAGE)
                    }
                }
            }
        }
    }
}

def updateArgoRepository(String appPath, String version, String argoBranch, String argoRepo) {
    withCredentials([usernamePassword(credentialsId: 'github-access', passwordVariable: 'password', usernameVariable: 'username')]) {
        if (!fileExists(argoRepo)){
            sh "git clone --branch $argoBranch https://${username}:${password}@github.com/empathyco/${argoRepo}.git"
        }
        dir( argoRepo ) {
            withCredentials([usernamePassword(credentialsId: 'harbor-credentials', usernameVariable: 'HELM_REPO_USERNAME', passwordVariable: 'HELM_REPO_PASSWORD')]) {

                sh "sed -i '1,/    version:/{s/version:.*/version: '${version}'/}' ${appPath}/Chart.yaml"
                sh "sed -i 's/appVersion: .*/appVersion: \"${version}\"/' ${appPath}/Chart.yaml"
                sh "helm repo add --username ${HELM_REPO_USERNAME} --password ${HELM_REPO_PASSWORD} empathy ${HELM_REPO}"
                sh "helm dep update ${appPath}"
                sh "git add ${appPath}"
                sh "git commit -am 'Update service: ${appPath} to version: ${version}'|| true"
                sh "git push origin $argoBranch"
            }
        }
    }
}

def isTaggedCommitOnMainBranch() {
    if(env.TAG_NAME == null) {
      return false;
    }

    branches = sh(returnStdout: true, script: "git branch --remote --no-color --contains tags/${env.TAG_NAME} | cut -c3-")
    return branches.split("\n").any{ branch -> branch.equals("origin/main") }
}