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
			Name:      RunnerJobName,
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
					ServiceAccountName: ServiceAccountName,
				},
			},
		},
	}
}

var (
	sa *corev1.ServiceAccount
)

func (i *Invocation) GetServiceAccountSpec() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: i.Namespace(),
		},
	}
}

func (i *TestOptions) CreateRunnerJob() error {
	sa = i.GetServiceAccountSpec()
	err := i.myClient.Create(context.TODO(), sa)
	if err != nil {
		return err
	}
	err = i.myClient.Create(context.TODO(), i.InitJob)
	return err
}

func (i *TestOptions) DeleteRunnerJob() error {
	err := i.myClient.Delete(context.TODO(), sa)
	if err != nil {
		return err
	}

	var pods corev1.PodList
	// list the pods & delete the one which is owned by the 'RunnerJob'
	err = i.myClient.List(context.TODO(), &pods)
	for _, pod := range pods.Items {
		owners := pod.OwnerReferences
		for _, owner := range owners {
			if owner.UID == i.InitJob.UID {
				err = i.myClient.Delete(context.TODO(), &pod)
				if err != nil {
					return err
				}
			}
		}
	}
	err = i.myClient.Delete(context.TODO(), i.InitJob)
	return err
}
