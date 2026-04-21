/*
Copyright 2024 Avodah Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main is the entrypoint for the conversation family member binary.
// It registers Conversation, ConversationBookmark, and ConversationPin controllers.
package main

import (
	"flag"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	bookmarkv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/bookmark/v1alpha1"
	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
	pinv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/pin/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/bookmark"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/conversation"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/pin"
)

func main() {
	var (
		debug          bool
		syncPeriod     time.Duration
		leaderElection bool
	)

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.DurationVar(&syncPeriod, "sync-period", 10*time.Hour, "Controller manager sync period")
	flag.BoolVar(&leaderElection, "leader-election", false, "Enable leader election")
	flag.Parse()

	zl := zap.New(zap.UseDevMode(debug))
	ctrl.SetLogger(zl)
	log := ctrl.Log.WithName("provider-conversation")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(conversationv1alpha1.AddToScheme(scheme))
	utilruntime.Must(bookmarkv1alpha1.AddToScheme(scheme))
	utilruntime.Must(pinv1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   leaderElection,
		LeaderElectionID: "provider-conversation.slack.crossplane.io",
		Cache: cache.Options{
			SyncPeriod: &syncPeriod,
		},
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	if err := conversation.Setup(mgr, controller.Options{}); err != nil {
		log.Error(err, "unable to setup Conversation controller")
		os.Exit(1)
	}

	if err := bookmark.Setup(mgr, controller.Options{}); err != nil {
		log.Error(err, "unable to setup ConversationBookmark controller")
		os.Exit(1)
	}

	if err := pin.Setup(mgr, controller.Options{}); err != nil {
		log.Error(err, "unable to setup ConversationPin controller")
		os.Exit(1)
	}

	log.Info("starting provider-conversation manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}
}
