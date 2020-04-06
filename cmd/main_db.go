package cmd

import (
	bolt "go.etcd.io/bbolt"
)

func saveToDB(pkg *Package, filter string, filename string, version string) error {
	err := DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(PackagesBucket)

		pb, err := bucket.CreateBucketIfNotExists([]byte(pkg.GetFullName()))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("filename"), []byte(filename))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("filter"), []byte(filter))
		if err != nil {
			return err
		}
		err = pb.Put([]byte("version"), []byte(version))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func loadInstalledFromDB() ([]*Package, error) {
	var pkgs []*Package
	err := DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(PackagesBucket))
		c := b.Cursor()
		for key, _ := c.First(); key != nil; key, _ = c.Next() {
			pb := b.Bucket(key)
			if pb != nil {
				p, err := CreatePackage(string(key))
				if err != nil {
					continue
				}
				p.Version = string(pb.Get([]byte("version")))
				p.Filter = string(pb.Get([]byte("filter")))
				pkgs = append(pkgs, p)
			}
		}
		return nil
	})
	return pkgs, err
}
