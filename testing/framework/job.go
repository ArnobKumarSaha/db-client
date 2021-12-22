package framework

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Invocation) GetTheRunnerJob() *batchv1.Job {

	return &batchv1.Job{
		ObjectMeta: meta.ObjectMeta{
			Name:      "kubernetes-go-test",
			Namespace: i.Namespace(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "kubernetes-go-test",
							Image:           "arnobkumarsaha/kubernetes-go-test",
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

func (i *Invocation) CreateRunnerJob(j *batchv1.Job) error {
	err := i.myClient.Create(context.TODO(), j)
	return err
}
func (i *Invocation) DeleteRunnerJob(j *batchv1.Job) error {
	err := i.myClient.Delete(context.TODO(), j)
	return err
}
