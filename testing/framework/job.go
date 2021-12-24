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
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: "",
				},
			},
		},
	}
}

func (i *TestOptions) CreateRunnerJob() error {
	err := i.myClient.Create(context.TODO(), i.InitJob)
	return err
}
func (i *TestOptions) DeleteRunnerJob() error {
	err := i.myClient.Delete(context.TODO(), i.InitJob)
	return err
}
