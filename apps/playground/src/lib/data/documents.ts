// Sample documents for semantic search simulation
// In production, these would be stored in a vector database like Pinecone, Weaviate, etc.

export interface Document {
  id: string;
  text: string;
  source: string;
  category: string;
}

export const sampleDocuments: Document[] = [
  // Policies
  {
    id: 'doc-1',
    text: 'Refund Policy: Full refunds are available within 30 days of purchase for unopened items. Opened items can be returned within 14 days for store credit. Defective products are eligible for free replacement or full refund at any time within the warranty period.',
    source: 'policies/refund-policy.md',
    category: 'policy',
  },
  {
    id: 'doc-2',
    text: 'Shipping Information: Free standard shipping on orders over $50. Express shipping (2-3 days) available for $9.99. Next-day delivery available in select areas for $19.99. International shipping rates vary by destination.',
    source: 'policies/shipping.md',
    category: 'policy',
  },
  {
    id: 'doc-3',
    text: 'Warranty Coverage: All products come with a 1-year limited warranty. AppleCare+ extends coverage to 2 years and includes accidental damage protection. Warranty claims can be filed online or at any authorized service provider.',
    source: 'policies/warranty.md',
    category: 'policy',
  },

  // Product Information
  {
    id: 'doc-4',
    text: 'MacBook Pro Technical Specs: The MacBook Pro 14" features the Apple M3 Pro chip with 12-core CPU and 18-core GPU. It includes 18GB unified memory, 512GB SSD storage, and a stunning Liquid Retina XDR display with ProMotion technology.',
    source: 'products/macbook-pro.md',
    category: 'product',
  },
  {
    id: 'doc-5',
    text: 'iPhone 15 Pro Features: Built with aerospace-grade titanium, the iPhone 15 Pro features the A17 Pro chip, a 48MP main camera with 5x optical zoom, and USB-C connectivity. Available in Natural Titanium, Blue Titanium, White Titanium, and Black Titanium.',
    source: 'products/iphone-15-pro.md',
    category: 'product',
  },
  {
    id: 'doc-6',
    text: 'AirPods Pro 2 Specifications: Second generation AirPods Pro feature H2 chip, Active Noise Cancellation, Adaptive Transparency, and Personalized Spatial Audio. Battery life: 6 hours of listening time with ANC on, up to 30 hours total with charging case.',
    source: 'products/airpods-pro.md',
    category: 'product',
  },

  // Support
  {
    id: 'doc-7',
    text: 'Troubleshooting Battery Drain: If your device battery is draining quickly, try these steps: 1) Check for background app refresh, 2) Reduce screen brightness, 3) Disable location services for non-essential apps, 4) Update to the latest software version.',
    source: 'support/battery-troubleshooting.md',
    category: 'support',
  },
  {
    id: 'doc-8',
    text: 'How to Reset Your Device: For a soft reset, press and hold the power button until the slider appears. For a hard reset, press and quickly release volume up, then volume down, then hold the side button until the Apple logo appears.',
    source: 'support/device-reset.md',
    category: 'support',
  },

  // FAQ
  {
    id: 'doc-9',
    text: 'Frequently Asked Questions: Q: Can I trade in my old device? A: Yes, we offer trade-in credit for eligible devices. Trade-in values depend on device condition and model. Q: Do you offer student discounts? A: Yes, verified students receive 10% off on select products.',
    source: 'faq/general.md',
    category: 'faq',
  },
  {
    id: 'doc-10',
    text: 'Payment Options: We accept all major credit cards, Apple Pay, PayPal, and financing through Apple Card. Monthly installments available with 0% APR for qualified customers. Gift cards can be applied to any purchase.',
    source: 'faq/payment.md',
    category: 'faq',
  },
];

// Simple keyword-based search (simulates semantic search for demo purposes)
// In production, you would use actual embeddings and vector similarity
export function searchDocuments(query: string, limit: number = 5): Array<Document & { score: number }> {
  const queryTerms = query.toLowerCase().split(/\s+/);

  const results = sampleDocuments.map(doc => {
    const text = doc.text.toLowerCase();
    let score = 0;

    for (const term of queryTerms) {
      if (text.includes(term)) {
        // Count occurrences
        const occurrences = (text.match(new RegExp(term, 'g')) || []).length;
        score += occurrences * 10;

        // Bonus for title/category match
        if (doc.source.toLowerCase().includes(term)) {
          score += 20;
        }
        if (doc.category.toLowerCase().includes(term)) {
          score += 15;
        }
      }
    }

    // Normalize score to 0-1 range
    const normalizedScore = Math.min(score / 100, 1);

    return { ...doc, score: normalizedScore };
  });

  return results
    .filter(r => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, limit);
}
