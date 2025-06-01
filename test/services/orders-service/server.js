const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const PORT = 8084;

app.use(bodyParser.json());

const orders = {
    'ord-001': {
        id: 'ord-001',
        userId: 'usr-001',
        total: 1199.98,
        date: '2023-05-15',
        status: 'delivered',
        items: [
            { productId: 'prod-001', quantity: 1, price: 999.99 },
            { productId: 'prod-003', quantity: 1, price: 199.99 }
        ]
    },
    'ord-002': {
        id: 'ord-002',
        userId: 'usr-002',
        total: 699.99,
        date: '2023-06-20',
        status: 'processing',
        items: [
            { productId: 'prod-002', quantity: 1, price: 699.99 }
        ]
    }
};

const authenticate = (req, res, next) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
        return res.status(401).json({ error: 'Authentication required' });
    }
    next();
};

// Get all orders
app.get('/api/orders', authenticate, (req, res) => {
    const userId = req.query.userId;
    if (userId) {
        const userOrders = Object.values(orders).filter(order => order.userId === userId);
        return res.json(userOrders);
    }
    res.json(Object.values(orders));
});

// Get order by ID
app.get('/api/orders/:id', authenticate, (req, res) => {
    const { id } = req.params;
    const order = orders[id];

    if (!order) {
        return res.status(404).json({ error: 'Order not found' });
    }

    res.json(order);
});

// Create new order
app.post('/api/orders', authenticate, (req, res) => {
    const { userId, items } = req.body;

    if (!userId || !items || !Array.isArray(items) || items.length === 0) {
        return res.status(400).json({ error: 'Invalid order data' });
    }

    const orderId = `ord-${Date.now()}`;
    const total = items.reduce((sum, item) => sum + (item.price * item.quantity), 0);

    const newOrder = {
        id: orderId,
        userId,
        items,
        total,
        date: new Date().toISOString().split('T')[0],
        status: 'processing'
    };

    orders[orderId] = newOrder;
    res.status(201).json(newOrder);
});

// Update order status
app.put('/api/orders/:id/status', authenticate, (req, res) => {
    const { id } = req.params;
    const { status } = req.body;

    if (!orders[id]) {
        return res.status(404).json({ error: 'Order not found' });
    }

    orders[id].status = status;
    res.json(orders[id]);
});

// Delete order
app.delete('/api/orders/:id', authenticate, (req, res) => {
    const { id } = req.params;

    if (!orders[id]) {
        return res.status(404).json({ error: 'Order not found' });
    }

    delete orders[id];
    res.status(204).end();
});

app.listen(PORT, () => {
    console.log(`Orders service running on port ${PORT}`);
});
