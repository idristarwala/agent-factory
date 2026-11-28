// Example Jenkins pipeline for Agent Factory (#9).
// Validates packs and runs offline fixture tests without live model keys.
// See docs/GETTING_STARTED.md and .gitlab-ci.yml for equivalents.

pipeline {
    agent {
        docker {
            image 'golang:1.26'
            args '-u root'
        }
    }
    stages {
        stage('Validate') {
            steps {
                sh 'go run ./cmd/factory validate examples/starter/factory.yaml || true'
                sh 'go run ./cmd/factory readiness || true'
            }
        }
        stage('Test') {
            steps {
                sh 'go run ./cmd/factory test'
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: '**/test-receipt*.json', allowEmptyArchive: true
        }
    }
}
