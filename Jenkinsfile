properties([[$class: 'BuildDiscarderProperty',
             strategy: [$class: 'LogRotator', numToKeepStr: '10']]])

def branchName = currentBuild.projectName

node {
    def image

    stage('Build container') {
        def scmVars = checkout scm
        def imageTag = branchName.equals("master") ? "latest" : scmVars.GIT_COMMIT
        image = docker.build("rafaelmartins/yatr:${imageTag}", '--no-cache --rm .')
    }

    if (branchName.equals('master')) {
        stage('Push container') {
            image.push()
        }
    }

    stage('Clean container') {
        sh "docker rmi ${image.id}"
    }
}
