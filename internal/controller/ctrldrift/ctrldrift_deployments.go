package ctrldrift

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func int32Ptr(i int32) *int32 {
	return &i
}

func get_converting_job() *batchv1.Job {
	converting_job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "converting-job",
			Labels: map[string]string{
				"app": "converting-lite",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(0),
			Completions:  int32Ptr(1),
			Parallelism:  int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "converting-lite",
							Image:           "lucaserf/converting-lite:latest",
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data-volume",
									MountPath: "/var/data/",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "FOLDER_PATH",
									Value: "/var/data/",
								},
								{
									Name:  "OUTPUT_PATH",
									Value: "regression_model_tf",
								},
								{
									Name:  "DATA_PATH",
									Value: "data.csv",
								},
								{
									Name:  "LOGGING_LEVEL",
									Value: "INFO",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "data-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	return converting_job
}

func get_training_job() *batchv1.Job {
	training_job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "training-job",
			Labels: map[string]string{
				"app": "training-regression",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(0),
			Completions:  int32Ptr(1),
			Parallelism:  int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "training-regression",
							Image: "lucaserf/training-regression:latest",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data-volume",
									MountPath: "/var/data/",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "FOLDER_PATH",
									Value: "/var/data/",
								},
								{
									Name:  "MODEL_PATH",
									Value: "regression_model_tf.keras",
								},
								{
									Name:  "OUTPUT_PATH",
									Value: "regressiong_model_lite",
								},
							},
						},
					},
					RestartPolicy: "Never",
					Volumes: []corev1.Volume{
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "data-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	return training_job
}

func get_drift_detection_deployment() *appsv1.Deployment {
	drift_detection_deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "drift-deploy",
			Labels: map[string]string{
				"app": "drift-detection",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "drift-detection",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "drift-detection",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "drift-detection",
							Image:           "lucaserf/drift_detection:latest",
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "FOLDER_PATH",
									Value: "/var/data/",
								},
								{
									Name:  "BROKER_ADDRESS",
									Value: "as-sensiblecity1.cloudmmwunibo.it",
								},
								{
									Name:  "TOPIC_NAME",
									Value: "drift-detection",
								},
								{
									Name:  "BATCH_SIZE",
									Value: "100",
								},
								{
									Name:  "ALPHA_P_VALUE",
									Value: "0.001",
								},
								{
									Name:  "OUTPUT_NAME",
									Value: "drift_data.csv",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data-volume",
									MountPath: "/var/data/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "data-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	return drift_detection_deployment
}
