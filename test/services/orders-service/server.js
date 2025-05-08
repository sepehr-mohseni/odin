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

app.get('/api/orders', authenticate, (req, res) => {
    res.json({ data: Object.values(orders) });
});

app.get('/api/orders/:id', authenticate, (req, res) => {
    const order = orders[req.params.id];
    if (!order) {
        return res.status(404).json({ error: 'Order not found' });
    }
    res.json(order);
});

app.post('/api/orders', authenticate, (req, res) => {
    const id = `ord-${Math.floor(Math.random() * 1000)}`;
    const newOrder = { id, ...req.body };
    orders[id] = newOrder;
    res.status(201).json(newOrder);
});

app.put('/api/orders/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!orders[id]) {
        return res.status(404).json({ error: 'Order not found' });
    }
    orders[id] = { ...orders[id], ...req.body };
    res.json(orders[id]);
});

app.delete('/api/orders/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!orders[id]) {
        return res.status(404).json({ error: 'Order not found' });
    }
    delete orders[id];
    res.status(204).end();
});

app.listen(PORT, () => {
    console.log(`Orders service running on port ${PORT}`);
});
