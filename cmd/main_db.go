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
