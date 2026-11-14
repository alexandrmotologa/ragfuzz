# Mathematical Metrics Guide

ragfuzz evaluates RAG pipelines across two distinct stages: the Retrieval Stage (retriever performance) and the Generation Stage (LLM output grounding and refusal calibration).

---

## 1. Retrieval Stage Metrics

### 1.1 Recall@K Under Adversarial Mutation
Given the set of golden supporting chunks $G_q$ for query $q$ and the top-$K$ retrieved chunks $R_{q,K}$:

$$\text{Recall@K}(q) = \frac{|R_{q,K} \cap G_q|}{|G_q|}$$

The **Recall Degradation** metric measures the mean drop in retrieval recall caused by adversarial perturbations across all test cases $Q$:

$$\Delta \text{Recall}_{\text{stress}} = \frac{1}{|Q|} \sum_{q \in Q} \left( \text{Recall@K}(q_{\text{clean}}) - \text{Recall@K}(q_{\text{mutated}}) \right)$$

### 1.2 Mean Reciprocal Rank (MRR)
Evaluates how high in the retrieved context the first true supporting chunk appears:

$$\text{MRR} = \frac{1}{|Q|} \sum_{q \in Q} \frac{1}{\text{rank}_q}$$

Where $\text{rank}_q$ is the 1-based index of the first ground-truth chunk in retrieved results, or 0 if no relevant chunk is returned.

### 1.3 Distractor Intrusion Rate
Measures the proportion of queries where a pre-identified distractor chunk $C_B$ successfully pushed into the top-$K$ retrieved results:

$$\text{Intrusion Rate} = \frac{\sum_{q \in Q} \mathbb{I}(R_{q,K} \cap D_q \neq \emptyset)}{|Q|}$$

---

## 2. Generation Stage Metrics

### 2.1 Groundedness (Faithfulness) Score
For a generated response $A$ decomposed into atomic factual claims $c_1, c_2, \dots, c_m$ and retrieved context text $C$:

$$\text{Groundedness}(A, C) = \frac{\sum_{i=1}^{m} \mathbb{I}(c_i \text{ is strictly entailed by } C)}{m}$$

Where entailment is determined by directional Natural Language Inference (NLI) verifying semantic keyword coverage, number preservation, and polarity alignment.

### 2.2 Hallucination Rate
The direct complement of Groundedness:

$$\text{Hallucination Rate} = 1.0 - \text{Groundedness}(A, C)$$

### 2.3 Refusal Calibration F1 ($F_1^{\text{refusal}}$)
Measures the ability to reject ungrounded out-of-domain queries ($Q_{\text{ood}}$) while continuing to answer valid in-domain queries ($Q_{\text{id}}$):

$$P_{\text{refusal}} = \frac{\text{True Refusals}}{\text{True Refusals} + \text{False Refusals}}$$

$$R_{\text{refusal}} = \frac{\text{True Refusals}}{|Q_{\text{ood}}|}$$

$$F_1^{\text{refusal}} = 2 \cdot \frac{P_{\text{refusal}} \cdot R_{\text{refusal}}}{P_{\text{refusal}} + R_{\text{refusal}}}$$
