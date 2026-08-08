#!/usr/bin/env python3
"""Trains a tiny TF-IDF + logistic-regression sentiment classifier and
exports it to ONNX for Triton Inference Server (CPU, onnxruntime backend).

This is intentionally small: a demo-scale labeled dataset (~80 short
examples), not a production sentiment model. The point of this script is
the *pipeline* — real training, real ONNX export (including the text
vectorizer inside the graph, so Triton takes raw strings as input), real
serving — which is what postsvc's ML enrichment step actually calls.
Swap in a larger dataset / different sklearn pipeline here without
touching anything downstream: the ONNX contract (string in, label +
probabilities out) stays the same.

Usage:
    pip install scikit-learn skl2onnx onnx
    python3 train_sentiment_model.py
"""
import numpy as np
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression
from sklearn.pipeline import Pipeline
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import StringTensorType

POSITIVE = [
    "I love this post, it's fantastic and made my day",
    "Great writeup, really helpful and well explained",
    "This is amazing work, thank you for sharing",
    "Wonderful article, learned so much from it",
    "Excellent explanation, very clear and useful",
    "Best post I've read all week, brilliant",
    "So happy I found this, exactly what I needed",
    "Fantastic tutorial, easy to follow and complete",
    "This made me smile, really enjoyable read",
    "Incredible insight, thank you so much for this",
    "Awesome content, keep up the great work",
    "I really appreciate this post, very well done",
    "Superb explanation, everything is clear now",
    "This is exactly what I was looking for, great job",
    "Delightful read, very well written and informative",
    "Love the detail here, top quality content",
    "Perfect timing, this solved my problem completely",
    "Really impressive work, congratulations on this",
    "Such a helpful guide, thanks a lot for sharing",
    "This post is pure gold, extremely valuable",
    "great post love it",
    "amazing thanks so much",
    "wonderful explanation really helpful",
    "fantastic work well done",
    "this is excellent and useful",
]

NEGATIVE = [
    "I hate this post, it's terrible and confusing",
    "Worst article I've read, complete waste of time",
    "This is awful, poorly written and useless",
    "Really disappointing, expected much better",
    "Terrible explanation, made things more confusing",
    "I regret reading this, total garbage content",
    "This made me angry, very frustrating experience",
    "Horrible tutorial, missing important steps",
    "This is boring and pointless, skip it",
    "Awful writeup, riddled with errors",
    "Bad content, not worth anyone's time",
    "I really dislike this post, badly organized",
    "Confusing explanation, nothing makes sense",
    "This is not what I needed, very unhelpful",
    "Disappointing read, poorly researched",
    "Hate the approach here, low quality content",
    "Bad timing, this made my problem worse",
    "Really unimpressive work, not recommended",
    "Such an unhelpful guide, thanks for nothing",
    "This post is a mess, extremely low value",
    "terrible post hate it",
    "awful waste of time",
    "bad explanation very confusing",
    "horrible work poorly done",
    "this is useless and bad",
]

NEUTRAL = [
    "This post covers three main topics in sequence",
    "The article was published on a Tuesday afternoon",
    "Here is a summary of the quarterly report",
    "The system processes requests using a queue",
    "This document describes the API endpoints available",
    "The meeting is scheduled for next Thursday",
    "Posts are stored in a database table",
    "The function accepts two parameters and returns a value",
    "This section explains the configuration options",
    "The server listens on port eight thousand eighty",
    "Data is synchronized every fifteen minutes",
    "The report contains twelve pages of analysis",
    "Users can filter posts by author or date",
    "The pipeline consists of four distinct stages",
    "This update changes the default timeout value",
    "The dataset includes records from the last year",
    "Configuration is read from environment variables",
    "The service exposes a REST API for clients",
    "This page lists all available commands",
    "The build process compiles and packages the binary",
    "a standard technical post about configuration",
    "the report was filed on schedule",
    "this describes the process step by step",
    "the data is stored and indexed",
    "a routine update to the documentation",
]

texts = [t.lower() for t in POSITIVE + NEGATIVE + NEUTRAL]
labels = (
    ["positive"] * len(POSITIVE)
    + ["negative"] * len(NEGATIVE)
    + ["neutral"] * len(NEUTRAL)
)

# lowercase=False is deliberate: TfidfVectorizer(lowercase=True) makes
# skl2onnx emit an ONNX StringNormalizer op, and onnxruntime's
# StringNormalizer requires a C++ locale (en_US.UTF-8) that the Triton
# server container doesn't have installed — model load fails with
# "Failed to construct locale". Lowercasing training text here and the
# query text client-side (internal/ml/triton.Client) gets the same
# behavior without that runtime dependency.
pipeline = Pipeline([
    ("tfidf", TfidfVectorizer(lowercase=False, ngram_range=(1, 2), max_features=2000)),
    ("clf", LogisticRegression(max_iter=1000, C=5.0)),
])
pipeline.fit(texts, labels)

train_acc = pipeline.score(texts, labels)
print(f"train accuracy: {train_acc:.3f}")

onnx_model = convert_sklearn(
    pipeline,
    "post_sentiment",
    initial_types=[("input", StringTensorType([None, 1]))],
    options={id(pipeline.named_steps["clf"]): {"zipmap": False}},
)

out_path = "model_repository/post_sentiment/1/model.onnx"
with open(out_path, "wb") as f:
    f.write(onnx_model.SerializeToString())
print(f"wrote {out_path}")

# Sanity-check with onnxruntime before handing off to Triton.
import onnxruntime as ort  # noqa: E402

sess = ort.InferenceSession(out_path, providers=["CPUExecutionProvider"])
sample = np.array([["i love this, fantastic work"]], dtype=object)
outputs = sess.run(None, {"input": sample})
print("sample prediction:", dict(zip([o.name for o in sess.get_outputs()], outputs)))
